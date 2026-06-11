package s3

import (
	"context"

	domains3 "aws-terminal/internal/domain/s3"
)

type ObjectStore interface {
	ListBuckets(ctx context.Context, profileName, region string) ([]domains3.Bucket, error)
	ListObjects(ctx context.Context, profileName, region, bucket, prefix string) ([]domains3.RemoteObject, error)
	UploadFile(ctx context.Context, input UploadFileInput) error
	DeleteObject(ctx context.Context, input DeleteObjectInput) error
	DeleteObjects(ctx context.Context, input DeleteObjectsInput) error
}

type UploadFileInput struct {
	Profile   string
	Region    string
	Bucket    string
	Key       string
	LocalPath string
	Metadata  domains3.UploadMetadata
	Progress  func(bytes int64)
}

type DeleteObjectInput struct {
	Profile string
	Region  string
	Bucket  string
	Key     string
}

type DeleteObjectsInput struct {
	Profile string
	Region  string
	Bucket  string
	Keys    []string
}
