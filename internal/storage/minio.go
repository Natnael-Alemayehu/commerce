package storage

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOClient wraps the minio-go client with commerce-specific operations.
type MinIOClient struct {
	client     *minio.Client
	bucket     string
	endpoint   string
	useSSL     bool
}

// NewMinIOClient creates a new MinIO client and ensures the bucket exists.
func NewMinIOClient(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*MinIOClient, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("check bucket exists: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("create bucket: %w", err)
		}
	}

	return &MinIOClient{
		client:   client,
		bucket:   bucket,
		endpoint: endpoint,
		useSSL:   useSSL,
	}, nil
}

// PresignedUploadURL generates a presigned PUT URL for direct client upload.
// The objectName should include a folder prefix, e.g. "products/uuid-filename.jpg".
func (m *MinIOClient) PresignedUploadURL(ctx context.Context, objectName string, expiry time.Duration) (*url.URL, error) {
	presignedURL, err := m.client.PresignedPutObject(ctx, m.bucket, objectName, expiry)
	if err != nil {
		return nil, fmt.Errorf("generate presigned upload url: %w", err)
	}
	return presignedURL, nil
}

// PublicURL returns the publicly accessible URL for an object.
// Assumes the bucket policy allows public read.
func (m *MinIOClient) PublicURL(objectName string) string {
	scheme := "http"
	if m.useSSL {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/%s/%s", scheme, m.endpoint, m.bucket, objectName)
}

// DeleteObject removes an object from the bucket.
func (m *MinIOClient) DeleteObject(ctx context.Context, objectName string) error {
	if err := m.client.RemoveObject(ctx, m.bucket, objectName, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("delete object: %w", err)
	}
	return nil
}
