package config

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
)

// KMSConfig はKMS設定を表す
type KMSConfig struct {
	Enabled     bool   `json:"enabled"`
	KeyID       string `json:"key_id"`
	Region      string `json:"region"`
	DataKeyPath string `json:"data_key_path"`
}

// Config はアプリケーションの設定を表す
type Config struct {
	AWSRegion string
	KMS       KMSConfig `json:"kms"`
}

// LoadConfig は環境変数から設定を読み込む
func LoadConfig() (*Config, error) {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "ap-northeast-1" // デフォルトリージョン
	}

	// KMS設定を環境変数から読み込み
	kmsKeyID := os.Getenv("KMS_KEY_ID")
	kmsRegion := os.Getenv("KMS_REGION")
	if kmsRegion == "" {
		kmsRegion = region // KMSリージョンが指定されていない場合はAWSリージョンを使用
	}
	kmsEnabled := kmsKeyID != "" // キーIDが設定されている場合は暗号化を有効にする
	dataKeyPath := os.Getenv("KMS_DATA_KEY_PATH")

	return &Config{
		AWSRegion: region,
		KMS: KMSConfig{
			Enabled:     kmsEnabled,
			KeyID:       kmsKeyID,
			Region:      kmsRegion,
			DataKeyPath: dataKeyPath,
		},
	}, nil
}

// ValidateAWSCredentials はAWS認証情報が有効かどうかを確認する
func ValidateAWSCredentials(ctx context.Context) error {
	_, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS credentials: %w", err)
	}
	return nil
}
