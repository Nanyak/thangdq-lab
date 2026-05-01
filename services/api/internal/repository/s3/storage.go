package s3

import (
	"context"
	stderrors "errors"
	"fmt"
	"io"
	"time"

	"github.com/Nanyak/thangdq-lab/internal/entity"
	"github.com/Nanyak/thangdq-lab/pkg/config"
	"github.com/Nanyak/thangdq-lab/pkg/errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3Storage struct {
	client *s3.Client
	bucket string
}

func NewS3Storage(cfg *config.S3Config) (*S3Storage, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(
		context.Background(),
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID,
				cfg.SecretAccessKey,
				"",
			),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	var clientOpts []func(*s3.Options)
	if cfg.Endpoint != "" {
		clientOpts = append(clientOpts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		})
	}

	client := s3.NewFromConfig(awsCfg, clientOpts...)

	return &S3Storage{
		client: client,
		bucket: cfg.Bucket,
	}, nil
}

func (s *S3Storage) Upload(ctx context.Context, key string, reader io.Reader, contentType string) error {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        reader,
		ContentType: aws.String(contentType),
	}

	_, err := s.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("%w: %v", errors.ErrUploadFailed, err)
	}

	return nil
}

func (s *S3Storage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	result, err := s.client.GetObject(ctx, input)
	if err != nil {
		var nsk *types.NoSuchKey
		if stderrors.As(err, &nsk) {
			return nil, fmt.Errorf("%w: %s", errors.ErrFileNotFound, key)
		}
		return nil, fmt.Errorf("%w: %v", errors.ErrDownloadFailed, err)
	}

	return result.Body, nil
}

func (s *S3Storage) Delete(ctx context.Context, key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	_, err := s.client.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("%w: %v", errors.ErrDeleteFailed, err)
	}

	return nil
}

func (s *S3Storage) List(ctx context.Context, prefix string) ([]*entity.File, error) {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	}

	result, err := s.client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("s3 list failed: %w", err)
	}

	files := make([]*entity.File, 0, len(result.Contents))
	for _, obj := range result.Contents {
		file := &entity.File{
			Key:          aws.ToString(obj.Key),
			Size:         aws.ToInt64(obj.Size),
			LastModified: aws.ToTime(obj.LastModified),
		}
		files = append(files, file)
	}

	return files, nil
}

func (s *S3Storage) GetURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s.client)

	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	result, err := presignClient.PresignGetObject(ctx, input, func(opts *s3.PresignOptions) {
		opts.Expires = expiration
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return result.URL, nil
}
