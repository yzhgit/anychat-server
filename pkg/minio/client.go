package minio

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Config MinIO configuration
type Config struct {
	Endpoint  string   `mapstructure:"endpoint"`
	AccessKey string   `mapstructure:"access_key"`
	SecretKey string   `mapstructure:"secret_key"`
	UseSSL    bool     `mapstructure:"use_ssl"`
	Buckets   []string `mapstructure:"buckets"`
}

// Client MinIO client wrapper
type Client struct {
	client  *minio.Client
	buckets []string
}

// NewClient creates a new MinIO client
func NewClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("minio config is nil")
	}

	// Initialize MinIO client
	minioClient, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	client := &Client{
		client:  minioClient,
		buckets: cfg.Buckets,
	}

	// Ensure all buckets exist
	ctx := context.Background()
	for _, bucket := range cfg.Buckets {
		exists, err := minioClient.BucketExists(ctx, bucket)
		if err != nil {
			return nil, fmt.Errorf("failed to check bucket %s: %w", bucket, err)
		}

		if !exists {
			err = minioClient.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
			if err != nil {
				return nil, fmt.Errorf("failed to create bucket %s: %w", bucket, err)
			}
		}
	}

	return client, nil
}

// PutObject uploads an object
func (c *Client) PutObject(ctx context.Context, bucketName, objectName, filePath string, contentType string) (minio.UploadInfo, error) {
	return c.client.FPutObject(ctx, bucketName, objectName, filePath, minio.PutObjectOptions{
		ContentType: contentType,
	})
}

// PutObjectReader uploads an object from reader
func (c *Client) PutObjectReader(ctx context.Context, bucketName, objectName string, reader interface{}, objectSize int64, contentType string) (minio.UploadInfo, error) {
	return c.client.PutObject(ctx, bucketName, objectName, reader.(interface {
		Read(p []byte) (n int, err error)
	}), objectSize, minio.PutObjectOptions{
		ContentType: contentType,
	})
}

// GetObject gets an object
func (c *Client) GetObject(ctx context.Context, bucketName, objectName string) (*minio.Object, error) {
	return c.client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
}

// RemoveObject deletes an object
func (c *Client) RemoveObject(ctx context.Context, bucketName, objectName string) error {
	return c.client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
}

// StatObject gets object metadata
func (c *Client) StatObject(ctx context.Context, bucketName, objectName string) (minio.ObjectInfo, error) {
	return c.client.StatObject(ctx, bucketName, objectName, minio.StatObjectOptions{})
}

// PresignedGetObject generates a download presigned URL
// Default expiration is 1 hour
func (c *Client) PresignedGetObject(ctx context.Context, bucketName, objectName string, expires time.Duration) (*url.URL, error) {
	if expires == 0 {
		expires = time.Hour
	}
	return c.client.PresignedGetObject(ctx, bucketName, objectName, expires, nil)
}

// PresignedPutObject generates an upload presigned URL
// Default expiration is 1 hour
func (c *Client) PresignedPutObject(ctx context.Context, bucketName, objectName string, expires time.Duration) (*url.URL, error) {
	if expires == 0 {
		expires = time.Hour
	}
	return c.client.PresignedPutObject(ctx, bucketName, objectName, expires)
}

// GetClient gets the underlying minio.Client
func (c *Client) GetClient() *minio.Client {
	return c.client
}

// ListBuckets lists all buckets
func (c *Client) ListBuckets(ctx context.Context) ([]minio.BucketInfo, error) {
	return c.client.ListBuckets(ctx)
}

// Close closes the connection (MinIO client does not need explicit closing)
func (c *Client) Close() error {
	return nil
}
