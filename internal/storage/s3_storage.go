package storage

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/takaishi/acb/internal/encryption"
)

// S3Storage はS3への保存を実装
type S3Storage struct {
	client     *s3.Client
	bucket     string
	usePrefix  bool
	backupData *BackupData
	encryptor  Encryptor
}

// NewS3Storage は新しいS3Storageを作成する
func NewS3Storage(ctx context.Context, bucket string) (*S3Storage, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	client := s3.NewFromConfig(cfg)

	bucketLocation, err := client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
		Bucket: &bucket,
	})
	if err != nil {
		return nil, err
	}
	if client.Options().Region != string(bucketLocation.LocationConstraint) {
		cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(string(bucketLocation.LocationConstraint)))
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
		}
		client = s3.NewFromConfig(cfg)
	}
	return &S3Storage{
		client:     client,
		bucket:     bucket,
		backupData: NewBackupData(),
	}, nil
}

// SaveJSON はJSONデータをS3に保存する
func (s *S3Storage) SaveJSON(ctx context.Context, prefix, userPoolID string, filename string, data interface{}) error {
	// JSONエンコード
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	// メモリ上にデータを保存
	if _, exists := s.backupData.Files[userPoolID]; !exists {
		s.backupData.Files[userPoolID] = make([]BackupFile, 0)
	}
	s.backupData.Files[userPoolID] = append(s.backupData.Files[userPoolID], BackupFile{
		Name:    filename,
		Content: jsonData,
	})

	return nil
}

// CreateTarGz はディレクトリをtar.gz形式で圧縮する
func (s *S3Storage) CreateTarGz(ctx context.Context, prefix string) error {
	// メモリ上でtar.gzを作成
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// 各ユーザープールのファイルを処理
	for userPoolID, files := range s.backupData.Files {
		for _, file := range files {
			// tarヘッダーを作成
			header := &tar.Header{
				Name:    filepath.Join(userPoolID, file.Name),
				Size:    int64(len(file.Content)),
				Mode:    0644,
				ModTime: time.Now(),
			}

			// tarヘッダーを書き込み
			if err := tw.WriteHeader(header); err != nil {
				return fmt.Errorf("failed to write tar header: %w", err)
			}

			// ファイルの内容を書き込み
			if _, err := tw.Write(file.Content); err != nil {
				return fmt.Errorf("failed to write file content: %w", err)
			}
		}
	}

	// ライターをクローズ
	if err := tw.Close(); err != nil {
		return fmt.Errorf("failed to close tar writer: %w", err)
	}
	if err := gw.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// 暗号化が有効な場合は暗号化してから保存
	finalData := buf.Bytes()
	if s.encryptor != nil {
		// データを暗号化
		encryptedData, err := s.encryptor.Encrypt(ctx, finalData)
		if err != nil {
			return fmt.Errorf("failed to encrypt data: %w", err)
		}

		// バイナリ形式にシリアライズ
		serializedData, err := encryption.SerializeEncryptedData(encryptedData)
		if err != nil {
			return fmt.Errorf("failed to serialize encrypted data: %w", err)
		}
		finalData = serializedData
	}

	// S3にアップロード
	return s.WriteFile(ctx, prefix, finalData)
}

// WriteFile はファイルをS3に保存する
func (s *S3Storage) WriteFile(ctx context.Context, key string, data []byte) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}
	return nil
}

// ReadFile はS3からファイルを読み込む
func (s *S3Storage) ReadFile(ctx context.Context, key string) ([]byte, error) {
	fmt.Printf("bucket: %s, key: %s\n", s.bucket, key)
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download from S3: %w", err)
	}
	defer output.Body.Close()

	data, err := io.ReadAll(output.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// 暗号化されているかチェックして復号化
	if s.encryptor != nil && isEncryptedData(data) {
		encryptedData, err := encryption.DeserializeEncryptedData(data)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize encrypted data: %w", err)
		}

		decryptedData, err := s.encryptor.Decrypt(ctx, encryptedData)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt data: %w", err)
		}
		return decryptedData, nil
	}

	return data, nil
}

// GetBackupData はバックアップデータを返す
func (s *S3Storage) GetBackupData() *BackupData {
	return s.backupData
}

// SetEncryptor は暗号化処理を設定する
func (s *S3Storage) SetEncryptor(encryptor Encryptor) {
	s.encryptor = encryptor
}
