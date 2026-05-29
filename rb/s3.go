// Package rb — S3 client wrapper.
// Mirrors the S3 service functions from services/amazonS3.js.
package rb

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Client wraps the AWS S3 SDK client with the operations used by the PDF pipeline.
type S3Client struct {
	client  *s3.Client
	presign *s3.PresignClient
	bucket  string
}

// NewS3Client creates an S3Client using the default AWS credential chain
// (environment variables, ~/.aws/credentials, IAM role, etc.)
func NewS3Client(ctx context.Context, bucket string) (*S3Client, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}
	c := s3.NewFromConfig(cfg)
	return &S3Client{
		client:  c,
		presign: s3.NewPresignClient(c),
		bucket:  bucket,
	}, nil
}

// UploadBuffer uploads a byte slice to S3 and returns the key.
// Mirrors uploadBufferToS3() from services/amazonS3.js.
func (c *S3Client) UploadBuffer(ctx context.Context, key, displayName string, data []byte) (string, error) {
	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:             aws.String(c.bucket),
		Key:                aws.String(key),
		Body:               bytes.NewReader(data),
		ContentType:        aws.String("application/pdf"),
		ContentDisposition: aws.String(fmt.Sprintf(`attachment; filename="%s"`, displayName)),
	})
	if err != nil {
		return "", fmt.Errorf("s3 PutObject(%s): %w", key, err)
	}
	return key, nil
}

// DownloadFile downloads a file from S3 and returns its bytes.
// Mirrors downloadSearchPackgFileFromS3() from services/amazonS3.js.
func (c *S3Client) DownloadFile(ctx context.Context, key string) ([]byte, error) {
	resp, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("s3 GetObject(%s): %w", key, err)
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// GetPresignedURL generates a time-limited pre-signed GET URL.
// Mirrors getS3PreviewUrl() from services/amazonS3.js.
func (c *S3Client) GetPresignedURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	req, err := c.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("presign GetObject(%s): %w", key, err)
	}
	return req.URL, nil
}

// DeleteObject removes a key from S3.
// Mirrors rollbackFromS3() from services/amazonS3.js.
func (c *S3Client) DeleteObject(ctx context.Context, key string) error {
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	return err
}
