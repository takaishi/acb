package aws

import (
	"context"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws"
)

// S3Client はS3操作のためのクライアントを表す
type S3Client struct {
	client *s3.Client
}

// NewS3Client は新しいS3Clientを作成する
func NewS3Client(ctx context.Context) (*S3Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	return &S3Client{
		client: s3.NewFromConfig(cfg),
	}, nil
}

// ListBackups は指定されたパターンに一致するバックアップの一覧を取得する
func (c *S3Client) ListBackups(ctx context.Context, bucket, prefix, pattern string) ([]string, error) {
	var backups []string
	var continuationToken *string

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	for {
		input := &s3.ListObjectsV2Input{
			Bucket:            &bucket,
			Prefix:            &prefix,
			ContinuationToken: continuationToken,
		}

		output, err := c.client.ListObjectsV2(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to get backup list: %w", err)
		}

		for _, obj := range output.Contents {
			// metadata.jsonファイルを探す
			if strings.HasSuffix(*obj.Key, "/metadata.json") {
				// ユーザープールIDを抽出
				parts := strings.Split(*obj.Key, "/")
				if len(parts) >= 2 {
					userPoolID := parts[len(parts)-2]
					if regex.MatchString(userPoolID) {
						backups = append(backups, path.Dir(*obj.Key))
					}
				}
			}
		}

		if output.IsTruncated == nil || !*output.IsTruncated {
			break
		}
		continuationToken = output.NextContinuationToken
	}

	return backups, nil
}

func (c *S3Client) GetObject(ctx context.Context, bucket, key string) ([]byte, error) {
	bucketLocation, err := c.GetBucketLocation(ctx, bucket)
	if err != nil {
		return nil, err
	}

	if c.client.Options().Region != bucketLocation {
		cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(bucketLocation))
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
		}
		c.client = s3.NewFromConfig(cfg)
	}

	fmt.Printf("region: %s, bucketLocation: %s\n", c.client.Options().Region, bucketLocation)

	getObjectOutput, err := c.client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}

	dataKey, err := io.ReadAll(getObjectOutput.Body)
	if err != nil {
		return nil, err
	}
	return dataKey, nil
}

// GetBucketLocation は指定されたバケットのリージョンを取得する
func (c *S3Client) GetBucketLocation(ctx context.Context, bucket string) (string, error) {
	input := &s3.GetBucketLocationInput{
		Bucket: &bucket,
	}

	output, err := c.client.GetBucketLocation(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to get bucket location: %w", err)
	}

	return string(output.LocationConstraint), nil
}

// GetKMSKeyRegion はKMSキーのARNからリージョンを取得する
func GetKMSKeyRegion(keyARN string) (string, error) {
	// KMSキーのARN形式: arn:aws:kms:region:account:key/key-id
	parts := strings.Split(keyARN, ":")
	if len(parts) < 4 {
		return "", fmt.Errorf("invalid KMS key ARN format: %s", keyARN)
	}

	// リージョンは4番目の要素
	region := parts[3]
	if region == "" {
		return "", fmt.Errorf("region not found in KMS key ARN: %s", keyARN)
	}

	return region, nil
}

// GetKMSKeyRegionFromKeyID はKMSキーIDからリージョンを取得する
func GetKMSKeyRegionFromKeyID(ctx context.Context, keyID string) (string, error) {
	// まずデフォルトのリージョンでKMSクライアントを作成
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	kmsClient := kms.NewFromConfig(cfg)

	// キーの情報を取得
	input := &kms.DescribeKeyInput{
		KeyId: &keyID,
	}

	output, err := kmsClient.DescribeKey(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to describe KMS key: %w", err)
	}

	// ARNからリージョンを抽出
	parts := strings.Split(*output.KeyMetadata.Arn, ":")
	if len(parts) < 4 {
		return "", fmt.Errorf("invalid KMS key ARN format: %s", *output.KeyMetadata.Arn)
	}

	return parts[3], nil
}
