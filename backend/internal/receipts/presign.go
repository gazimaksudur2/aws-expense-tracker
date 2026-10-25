package receipts

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const DefaultExpiry = 5 * time.Minute

type Presigner struct {
	client *s3.PresignClient
	bucket string
}

func NewPresigner(ctx context.Context, bucket, endpoint string) (*Presigner, error) {
	if bucket == "" {
		return nil, fmt.Errorf("RECEIPTS_BUCKET is required")
	}
	opts := []func(*config.LoadOptions) error{}
	if endpoint != "" {
		region := os.Getenv("AWS_REGION")
		if region == "" {
			region = "us-east-1"
		}
		opts = append(opts,
			config.WithRegion(region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
		)
	}
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true // required for LocalStack
		}
	})
	return &Presigner{client: s3.NewPresignClient(client), bucket: bucket}, nil
}

func ObjectKey(userID, expenseID, filename string) string {
	base := filepath.Base(filename)
	base = strings.ReplaceAll(base, "..", "")
	return fmt.Sprintf("receipts/%s/%s/%s", userID, expenseID, base)
}

func (p *Presigner) PresignPut(ctx context.Context, key, contentType string) (url string, expiresIn int, err error) {
	out, err := p.client.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(p.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(DefaultExpiry))
	if err != nil {
		return "", 0, err
	}
	return out.URL, int(DefaultExpiry.Seconds()), nil
}
