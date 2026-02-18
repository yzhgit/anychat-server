package minio

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Config MinIO配置
type Config struct {
	Endpoint  string   `mapstructure:"endpoint"`
	AccessKey string   `mapstructure:"access_key"`
	SecretKey string   `mapstructure:"secret_key"`
	UseSSL    bool     `mapstructure:"use_ssl"`
	Buckets   []string `mapstructure:"buckets"`
}

// Client MinIO客户端封装
type Client struct {
	client  *minio.Client
	buckets []string
}

// NewClient 创建MinIO客户端
func NewClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("minio config is nil")
	}

	// 初始化MinIO客户端
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

	// 确保所有bucket存在
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

// PutObject 上传对象
func (c *Client) PutObject(ctx context.Context, bucketName, objectName, filePath string, contentType string) (minio.UploadInfo, error) {
	return c.client.FPutObject(ctx, bucketName, objectName, filePath, minio.PutObjectOptions{
		ContentType: contentType,
	})
}

// PutObjectReader 从Reader上传对象
func (c *Client) PutObjectReader(ctx context.Context, bucketName, objectName string, reader interface{}, objectSize int64, contentType string) (minio.UploadInfo, error) {
	return c.client.PutObject(ctx, bucketName, objectName, reader.(interface {
		Read(p []byte) (n int, err error)
	}), objectSize, minio.PutObjectOptions{
		ContentType: contentType,
	})
}

// GetObject 获取对象
func (c *Client) GetObject(ctx context.Context, bucketName, objectName string) (*minio.Object, error) {
	return c.client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
}

// RemoveObject 删除对象
func (c *Client) RemoveObject(ctx context.Context, bucketName, objectName string) error {
	return c.client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
}

// StatObject 获取对象元数据
func (c *Client) StatObject(ctx context.Context, bucketName, objectName string) (minio.ObjectInfo, error) {
	return c.client.StatObject(ctx, bucketName, objectName, minio.StatObjectOptions{})
}

// PresignedGetObject 生成下载presigned URL
// 默认有效期1小时
func (c *Client) PresignedGetObject(ctx context.Context, bucketName, objectName string, expires time.Duration) (*url.URL, error) {
	if expires == 0 {
		expires = time.Hour
	}
	return c.client.PresignedGetObject(ctx, bucketName, objectName, expires, nil)
}

// PresignedPutObject 生成上传presigned URL
// 默认有效期1小时
func (c *Client) PresignedPutObject(ctx context.Context, bucketName, objectName string, expires time.Duration) (*url.URL, error) {
	if expires == 0 {
		expires = time.Hour
	}
	return c.client.PresignedPutObject(ctx, bucketName, objectName, expires)
}

// GetClient 获取底层minio.Client
func (c *Client) GetClient() *minio.Client {
	return c.client
}

// ListBuckets 列出所有bucket
func (c *Client) ListBuckets(ctx context.Context) ([]minio.BucketInfo, error) {
	return c.client.ListBuckets(ctx)
}

// Close 关闭连接（MinIO客户端无需显式关闭）
func (c *Client) Close() error {
	return nil
}
