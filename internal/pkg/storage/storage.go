package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	CarPhotos       BucketConfig
	AccountPhotos   BucketConfig
	ModFiles        BucketConfig
}

type BucketConfig struct {
	Name      string
	PublicURL string
}

type Client struct {
	s3            *s3.Client
	carPhotos     BucketConfig
	accountPhotos BucketConfig
	modFiles      BucketConfig
}

func New(cfg Config) (*Client, error) {
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID)

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")),
		awsconfig.WithRegion("auto"),
	)
	if err != nil {
		return nil, fmt.Errorf("loading R2 config: %w", err)
	}

	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})

	return &Client{
		s3:            s3Client,
		carPhotos:     cfg.CarPhotos,
		accountPhotos: cfg.AccountPhotos,
		modFiles:      cfg.ModFiles,
	}, nil
}

func (c *Client) PresignCarPhotoUpload(ctx context.Context, key, contentType string) (string, error) {
	return c.presignPut(ctx, c.carPhotos.Name, key, contentType)
}

func (c *Client) PresignAccountPhotoUpload(ctx context.Context, key, contentType string) (string, error) {
	return c.presignPut(ctx, c.accountPhotos.Name, key, contentType)
}

func (c *Client) CarPhotoURL(key string) string {
	return c.carPhotos.PublicURL + "/" + key
}

func (c *Client) AccountPhotoURL(key string) string {
	return c.accountPhotos.PublicURL + "/" + key
}

func (c *Client) DeleteCarPhoto(ctx context.Context, key string) error {
	return c.deleteObject(ctx, c.carPhotos.Name, key)
}

func (c *Client) DeleteAccountPhoto(ctx context.Context, key string) error {
	return c.deleteObject(ctx, c.accountPhotos.Name, key)
}

func (c *Client) PresignModFileUpload(ctx context.Context, key, contentType string) (string, error) {
	return c.presignPut(ctx, c.modFiles.Name, key, contentType)
}

func (c *Client) ModFileURL(key string) string {
	return c.modFiles.PublicURL + "/" + key
}

func (c *Client) DeleteModFile(ctx context.Context, key string) error {
	return c.deleteObject(ctx, c.modFiles.Name, key)
}

func (c *Client) deleteObject(ctx context.Context, bucket, key string) error {
	_, err := c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	return err
}

func (c *Client) presignPut(ctx context.Context, bucket, key, contentType string) (string, error) {
	presigner := s3.NewPresignClient(c.s3)
	req, err := presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(15*time.Minute))
	if err != nil {
		return "", fmt.Errorf("presigning upload: %w", err)
	}
	return req.URL, nil
}
