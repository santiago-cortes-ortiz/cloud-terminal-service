package awss3

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3types "github.com/aws/aws-sdk-go-v2/service/s3/types"

	apps3 "aws-terminal/internal/application/s3"
	domains3 "aws-terminal/internal/domain/s3"
)

func TestBucketRegionCacheKeyIncludesProfileAndBucket(t *testing.T) {
	got := bucketRegionCacheKey(" prod ", " assets-bucket ")
	want := "prod|assets-bucket"
	if got != want {
		t.Fatalf("bucketRegionCacheKey() = %q, want %q", got, want)
	}
}

func TestBucketRegionCacheKeySeparatesProfiles(t *testing.T) {
	pre := bucketRegionCacheKey("pre", "shared-name")
	prod := bucketRegionCacheKey("prod", "shared-name")
	if pre == prod {
		t.Fatalf("expected different cache keys for different profiles, both got %q", pre)
	}
}

func TestCompactObjectKeysTrimsAndRemovesEmptyKeys(t *testing.T) {
	got := compactObjectKeys([]string{" index.html ", "", "  ", "assets/app.js"})
	want := []string{"index.html", "assets/app.js"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("compactObjectKeys() = %#v, want %#v", got, want)
	}
}

func TestDeleteObjectsRequestMapsBatchInput(t *testing.T) {
	request := deleteObjectsRequest(apps3.DeleteObjectsInput{Bucket: " bucket "}, []string{"old-a.js", "old-b.js"})

	if request.Bucket == nil || *request.Bucket != "bucket" {
		t.Fatalf("Bucket = %#v, want bucket", request.Bucket)
	}
	if request.Delete == nil {
		t.Fatal("expected Delete payload")
	}
	if request.Delete.Quiet == nil || !*request.Delete.Quiet {
		t.Fatalf("Quiet = %#v, want true", request.Delete.Quiet)
	}
	if len(request.Delete.Objects) != 2 {
		t.Fatalf("objects = %d, want 2", len(request.Delete.Objects))
	}
	if got := aws.ToString(request.Delete.Objects[0].Key); got != "old-a.js" {
		t.Fatalf("first key = %q", got)
	}
	if got := aws.ToString(request.Delete.Objects[1].Key); got != "old-b.js" {
		t.Fatalf("second key = %q", got)
	}
}

func TestDeleteObjectsErrorSummaryIncludesFirstErrorAndCount(t *testing.T) {
	summary := deleteObjectsErrorSummary([]awss3types.Error{
		{Key: aws.String("old.js"), Code: aws.String("AccessDenied"), Message: aws.String("denied")},
		{Key: aws.String("other.js"), Code: aws.String("AccessDenied")},
	})
	want := "old.js: AccessDenied - denied (+1 more)"
	if summary != want {
		t.Fatalf("deleteObjectsErrorSummary() = %q, want %q", summary, want)
	}
}

func TestProgressReaderReportsBytesRead(t *testing.T) {
	var reported []int64
	reader := &progressReader{
		reader: strings.NewReader("hello"),
		progress: func(bytes int64) {
			reported = append(reported, bytes)
		},
	}

	buf := make([]byte, 2)
	for {
		_, err := reader.Read(buf)
		if err != nil {
			break
		}
	}

	var total int64
	for _, bytes := range reported {
		total += bytes
	}
	if total != 5 {
		t.Fatalf("reported bytes = %d, want 5", total)
	}
}

func TestShouldUseMultipartUploadAtThreshold(t *testing.T) {
	if shouldUseMultipartUpload(multipartUploadThreshold - 1) {
		t.Fatal("expected size below threshold to use regular put object")
	}
	if !shouldUseMultipartUpload(multipartUploadThreshold) {
		t.Fatal("expected size at threshold to use multipart upload")
	}
}

func TestPutObjectInputSetsContentTypeAndMetadata(t *testing.T) {
	request := putObjectInput(apps3.UploadFileInput{
		Bucket:    " bucket ",
		Key:       " key ",
		LocalPath: "index.html",
		Metadata: domains3.UploadMetadata{
			CacheControl:    "no-cache",
			ContentEncoding: "gzip",
		},
	}, strings.NewReader("hello"), 5)

	if request.Bucket == nil || *request.Bucket != "bucket" {
		t.Fatalf("unexpected bucket %#v", request.Bucket)
	}
	if request.Key == nil || *request.Key != "key" {
		t.Fatalf("unexpected key %#v", request.Key)
	}
	if request.ContentType == nil || *request.ContentType != "text/html; charset=utf-8" {
		t.Fatalf("unexpected content type %#v", request.ContentType)
	}
	if request.ContentLength == nil || *request.ContentLength != 5 {
		t.Fatalf("unexpected content length %#v", request.ContentLength)
	}
	if request.CacheControl == nil || *request.CacheControl != "no-cache" {
		t.Fatalf("unexpected cache-control %#v", request.CacheControl)
	}
	if request.ContentEncoding == nil || *request.ContentEncoding != "gzip" {
		t.Fatalf("unexpected content-encoding %#v", request.ContentEncoding)
	}
}

func TestContentTypeForPathUsesMimePackageFirst(t *testing.T) {
	got := contentTypeForPath("index.HTML")
	want := "text/html; charset=utf-8"
	if got != want {
		t.Fatalf("contentTypeForPath() = %q, want %q", got, want)
	}
}

func TestContentTypeForPathFrontendFallbacks(t *testing.T) {
	tests := map[string]string{
		"site.webmanifest":     "application/manifest+json",
		"main.js.map":          "application/json",
		"favicon.ico":          "image/x-icon",
		"feed.xml":             "application/xml",
		"robots.txt":           "text/plain; charset=utf-8",
		"README.md":            "text/markdown; charset=utf-8",
		"image.avif":           "image/avif",
		"image.webp":           "image/webp",
		"font.woff":            "font/woff",
		"font.woff2":           "font/woff2",
		"font.ttf":             "font/ttf",
		"font.otf":             "font/otf",
		"without-extension":    "",
		"unknown.frontendtype": "",
	}

	for filePath, want := range tests {
		if got := contentTypeForPath(filePath); got != want {
			t.Errorf("contentTypeForPath(%q) = %q, want %q", filePath, got, want)
		}
	}
}
