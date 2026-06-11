package awss3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	awss3sdk "github.com/aws/aws-sdk-go-v2/service/s3"
	awss3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"

	apps3 "aws-terminal/internal/application/s3"
	domains3 "aws-terminal/internal/domain/s3"
	"aws-terminal/internal/infrastructure/awsclients"
)

const (
	multipartUploadThreshold   int64 = 100 * 1024 * 1024
	multipartUploadPartSize    int64 = 16 * 1024 * 1024
	multipartUploadConcurrency       = 4
)

type Store struct {
	mu            sync.RWMutex
	bucketRegions map[string]string
	clients       *awsclients.Factory
}

func NewStore() *Store {
	return NewStoreWithFactory(awsclients.Default())
}

func NewStoreWithFactory(clients *awsclients.Factory) *Store {
	if clients == nil {
		clients = awsclients.Default()
	}
	return &Store{
		bucketRegions: make(map[string]string),
		clients:       clients,
	}
}

func (s *Store) ListBuckets(ctx context.Context, profileName, region string) ([]domains3.Bucket, error) {
	ctx, cancel := awsclients.WithTimeout(ctx, s.clients.OperationTimeout())
	defer cancel()

	client, err := s.client(ctx, profileName, region)
	if err != nil {
		return nil, err
	}

	output, err := client.ListBuckets(ctx, &awss3sdk.ListBucketsInput{})
	if err != nil {
		return nil, err
	}

	buckets := make([]domains3.Bucket, 0, len(output.Buckets))
	for _, bucket := range output.Buckets {
		buckets = append(buckets, domains3.Bucket{
			Name:         aws.ToString(bucket.Name),
			CreationDate: aws.ToTime(bucket.CreationDate),
		})
	}

	return buckets, nil
}

func (s *Store) ListObjects(ctx context.Context, profileName, region, bucket, prefix string) ([]domains3.RemoteObject, error) {
	ctx, cancel := awsclients.WithTimeout(ctx, s.clients.OperationTimeout())
	defer cancel()

	client, err := s.bucketClient(ctx, profileName, region, bucket)
	if err != nil {
		return nil, err
	}

	paginator := awss3sdk.NewListObjectsV2Paginator(client, &awss3sdk.ListObjectsV2Input{
		Bucket: aws.String(strings.TrimSpace(bucket)),
		Prefix: aws.String(strings.TrimSpace(prefix)),
	})

	objects := make([]domains3.RemoteObject, 0)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, object := range page.Contents {
			objects = append(objects, domains3.RemoteObject{
				Key:  aws.ToString(object.Key),
				Size: aws.ToInt64(object.Size),
			})
		}
	}

	return objects, nil
}

func (s *Store) UploadFile(ctx context.Context, input apps3.UploadFileInput) error {
	ctx, cancel := awsclients.WithTimeout(ctx, s.clients.UploadTimeout())
	defer cancel()

	client, err := s.bucketClient(ctx, input.Profile, input.Region, input.Bucket)
	if err != nil {
		return err
	}

	file, err := os.Open(input.LocalPath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	request := putObjectInput(input, file, info.Size())
	if shouldUseMultipartUpload(info.Size()) {
		uploader := s3manager.NewUploader(client, func(u *s3manager.Uploader) {
			u.PartSize = multipartUploadPartSize
			u.Concurrency = multipartUploadConcurrency
		})
		_, err = uploader.Upload(ctx, request)
		return err
	}

	_, err = client.PutObject(ctx, request)
	return err
}

type progressReader struct {
	reader   io.Reader
	progress func(int64)
}

func (r *progressReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 && r.progress != nil {
		r.progress(int64(n))
	}
	return n, err
}

func putObjectInput(input apps3.UploadFileInput, body io.Reader, size int64) *awss3sdk.PutObjectInput {
	if input.Progress != nil {
		body = &progressReader{reader: body, progress: input.Progress}
	}

	request := &awss3sdk.PutObjectInput{
		Bucket:        aws.String(strings.TrimSpace(input.Bucket)),
		Key:           aws.String(strings.TrimSpace(input.Key)),
		Body:          body,
		ContentLength: aws.Int64(size),
	}
	if contentType := contentTypeForPath(input.LocalPath); contentType != "" {
		request.ContentType = aws.String(contentType)
	}
	if cacheControl := strings.TrimSpace(input.Metadata.CacheControl); cacheControl != "" {
		request.CacheControl = aws.String(cacheControl)
	}
	if contentEncoding := strings.TrimSpace(input.Metadata.ContentEncoding); contentEncoding != "" {
		request.ContentEncoding = aws.String(contentEncoding)
	}
	return request
}

func shouldUseMultipartUpload(size int64) bool {
	return size >= multipartUploadThreshold
}

func (s *Store) DeleteObject(ctx context.Context, input apps3.DeleteObjectInput) error {
	ctx, cancel := awsclients.WithTimeout(ctx, s.clients.OperationTimeout())
	defer cancel()

	client, err := s.bucketClient(ctx, input.Profile, input.Region, input.Bucket)
	if err != nil {
		return err
	}

	_, err = client.DeleteObject(ctx, &awss3sdk.DeleteObjectInput{
		Bucket: aws.String(strings.TrimSpace(input.Bucket)),
		Key:    aws.String(strings.TrimSpace(input.Key)),
	})
	return err
}

func (s *Store) DeleteObjects(ctx context.Context, input apps3.DeleteObjectsInput) error {
	ctx, cancel := awsclients.WithTimeout(ctx, s.clients.OperationTimeout())
	defer cancel()

	keys := compactObjectKeys(input.Keys)
	if len(keys) == 0 {
		return nil
	}

	client, err := s.bucketClient(ctx, input.Profile, input.Region, input.Bucket)
	if err != nil {
		return err
	}

	output, err := client.DeleteObjects(ctx, deleteObjectsRequest(input, keys))
	if err != nil {
		return err
	}
	if len(output.Errors) > 0 {
		return fmt.Errorf("delete objects failed: %s", deleteObjectsErrorSummary(output.Errors))
	}

	return nil
}

func deleteObjectsRequest(input apps3.DeleteObjectsInput, keys []string) *awss3sdk.DeleteObjectsInput {
	objects := make([]awss3types.ObjectIdentifier, 0, len(keys))
	for _, key := range keys {
		objects = append(objects, awss3types.ObjectIdentifier{Key: aws.String(key)})
	}

	return &awss3sdk.DeleteObjectsInput{
		Bucket: aws.String(strings.TrimSpace(input.Bucket)),
		Delete: &awss3types.Delete{
			Objects: objects,
			Quiet:   aws.Bool(true),
		},
	}
}

func compactObjectKeys(keys []string) []string {
	compacted := make([]string, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key != "" {
			compacted = append(compacted, key)
		}
	}
	return compacted
}

func deleteObjectsErrorSummary(errors []awss3types.Error) string {
	if len(errors) == 0 {
		return "unknown error"
	}

	first := errors[0]
	summary := strings.TrimSpace(aws.ToString(first.Key))
	if code := strings.TrimSpace(aws.ToString(first.Code)); code != "" {
		if summary != "" {
			summary += ": "
		}
		summary += code
	}
	if message := strings.TrimSpace(aws.ToString(first.Message)); message != "" {
		if summary != "" {
			summary += " - "
		}
		summary += message
	}
	if summary == "" {
		summary = "unknown error"
	}
	if len(errors) > 1 {
		summary += fmt.Sprintf(" (+%d more)", len(errors)-1)
	}
	return summary
}

func contentTypeForPath(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == "" {
		return ""
	}

	if contentType, ok := frontendContentTypes[ext]; ok {
		return contentType
	}

	if contentType := mime.TypeByExtension(ext); contentType != "" {
		return contentType
	}

	return ""
}

var frontendContentTypes = map[string]string{
	".webmanifest": "application/manifest+json",
	".map":         "application/json",
	".ico":         "image/x-icon",
	".xml":         "application/xml",
	".txt":         "text/plain; charset=utf-8",
	".md":          "text/markdown; charset=utf-8",
	".avif":        "image/avif",
	".webp":        "image/webp",
	".woff":        "font/woff",
	".woff2":       "font/woff2",
	".ttf":         "font/ttf",
	".otf":         "font/otf",
}

func (s *Store) client(ctx context.Context, profileName, region string) (*awss3sdk.Client, error) {
	client, err := s.clients.S3(ctx, profileName, region)
	if err != nil {
		return nil, fmt.Errorf("load S3 client: %w", err)
	}
	return client, nil
}

func (s *Store) bucketClient(ctx context.Context, profileName, region, bucket string) (*awss3sdk.Client, error) {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return nil, fmt.Errorf("bucket name is required")
	}

	region = awsclients.NormalizeRegion(region)

	client, err := s.client(ctx, profileName, region)
	if err != nil {
		return nil, err
	}

	bucketRegion, err := s.bucketRegion(ctx, profileName, client, bucket)
	if err != nil {
		return nil, err
	}
	if bucketRegion == "" || bucketRegion == region {
		return client, nil
	}

	return s.client(ctx, profileName, bucketRegion)
}

func (s *Store) bucketRegion(ctx context.Context, profileName string, client *awss3sdk.Client, bucket string) (string, error) {
	cacheKey := bucketRegionCacheKey(profileName, bucket)
	s.mu.RLock()
	region, ok := s.bucketRegions[cacheKey]
	s.mu.RUnlock()
	if ok {
		return region, nil
	}

	region, err := headBucketRegion(ctx, client, bucket)
	if err != nil {
		return "", fmt.Errorf("resolve region for bucket %q: %w", bucket, err)
	}

	s.mu.Lock()
	s.bucketRegions[cacheKey] = region
	s.mu.Unlock()
	return region, nil
}

func bucketRegionCacheKey(profileName, bucket string) string {
	return strings.TrimSpace(profileName) + "|" + strings.TrimSpace(bucket)
}

const bucketRegionHeader = "X-Amz-Bucket-Region"

type bucketRegionMiddleware struct {
	region string
}

func (m *bucketRegionMiddleware) ID() string {
	return "aws-terminal-bucket-region"
}

func (m *bucketRegionMiddleware) HandleDeserialize(ctx context.Context, in middleware.DeserializeInput, next middleware.DeserializeHandler) (
	out middleware.DeserializeOutput, metadata middleware.Metadata, err error,
) {
	out, metadata, err = next.HandleDeserialize(ctx, in)
	if response, ok := out.RawResponse.(*smithyhttp.Response); ok {
		m.region = strings.TrimSpace(response.Header.Get(bucketRegionHeader))
	}
	return out, metadata, err
}

func headBucketRegion(ctx context.Context, client *awss3sdk.Client, bucket string) (string, error) {
	capture := &bucketRegionMiddleware{}
	_, err := client.HeadBucket(ctx, &awss3sdk.HeadBucketInput{Bucket: aws.String(bucket)}, func(options *awss3sdk.Options) {
		options.APIOptions = append(options.APIOptions, func(stack *middleware.Stack) error {
			return stack.Deserialize.Add(capture, middleware.After)
		})
	})
	if capture.region != "" {
		return capture.region, nil
	}
	if err == nil {
		return "", nil
	}

	var statusError interface{ HTTPStatusCode() int }
	if errors.As(err, &statusError) && statusError.HTTPStatusCode() == http.StatusNotFound {
		return "", fmt.Errorf("bucket not found")
	}

	return "", err
}
