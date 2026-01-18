package storage

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	appcfg "github.com/jorgepascosoto/auto-db-backups/internal/config"
	"github.com/jorgepascosoto/auto-db-backups/internal/errors"
)

type R2Client struct {
	client    *s3.Client
	bucket    string
	prefix    string
	accountID string
}

type BackupObject struct {
	Key          string
	Size         int64
	LastModified time.Time
}

// NewR2Client creates a new R2 client with the specified prefix
func NewR2Client(ctx context.Context, cfg *appcfg.Config, prefix string) (*R2Client, error) {
	// Create custom HTTP client with explicit TLS configuration
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
				MaxVersion: tls.VersionTLS13,
			},
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
			IdleConnTimeout:       90 * time.Second,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
		},
		Timeout: 5 * time.Minute,
	}

	// Use the standard AWS configuration with custom HTTP client
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.R2AccessKeyID,
			cfg.R2SecretAccessKey,
			"",
		)),
		config.WithRegion("auto"),
		config.WithHTTPClient(httpClient),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client with R2 endpoint
	// R2 requires path-style addressing (bucket in path, not hostname)
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.R2AccountID))
		o.UsePathStyle = true
	})

	return &R2Client{
		client:    client,
		bucket:    cfg.R2BucketName,
		prefix:    prefix,
		accountID: cfg.R2AccountID,
	}, nil
}

func (c *R2Client) Upload(ctx context.Context, key string, body io.Reader) error {
	fullKey := c.prefix + key

	// Use the upload manager for better retry handling and large file support
	uploader := manager.NewUploader(c.client)

	_, err := uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(fullKey),
		Body:   body,
	})
	if err != nil {
		return errors.NewStorageError("upload", c.bucket, fullKey, err)
	}

	return nil
}

func (c *R2Client) Delete(ctx context.Context, key string) error {
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return errors.NewStorageError("delete", c.bucket, key, err)
	}

	return nil
}

func (c *R2Client) ListBackups(ctx context.Context) ([]BackupObject, error) {
	var backups []BackupObject

	paginator := s3.NewListObjectsV2Paginator(c.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(c.bucket),
		Prefix: aws.String(c.prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.NewStorageError("list", c.bucket, c.prefix, err)
		}

		for _, obj := range page.Contents {
			backups = append(backups, BackupObject{
				Key:          aws.ToString(obj.Key),
				Size:         aws.ToInt64(obj.Size),
				LastModified: aws.ToTime(obj.LastModified),
			})
		}
	}

	// Sort by last modified (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].LastModified.After(backups[j].LastModified)
	})

	return backups, nil
}

func (c *R2Client) Bucket() string {
	return c.bucket
}

func (c *R2Client) Prefix() string {
	return c.prefix
}
