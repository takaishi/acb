package storage

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/takaishi/acb/internal/encryption"
)

// LocalStorage はローカルファイルシステムへの保存を実装
type LocalStorage struct {
	backupData *BackupData
	encryptor  Encryptor
}

// NewLocalStorage は新しいLocalStorageを作成する
func NewLocalStorage() (*LocalStorage, error) {
	return &LocalStorage{
		backupData: NewBackupData(),
	}, nil
}

// SaveJSON はJSONデータをローカルファイルシステムに保存する
func (s *LocalStorage) SaveJSON(ctx context.Context, prefix, userPoolID string, filename string, data interface{}) error {
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

// ReadFile はローカルファイルシステムからファイルを読み込む
func (s *LocalStorage) ReadFile(ctx context.Context, key string) ([]byte, error) {
	data, err := os.ReadFile(key)
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

// CreateTarGz はディレクトリをtar.gz形式で圧縮する
func (s *LocalStorage) CreateTarGz(ctx context.Context, path string) error {
	fmt.Printf("path: %s\n", path)
	// file:// スキームを除去
	path = strings.TrimPrefix(path, "file://")
	// .tar.gz 拡張子を除去
	path = strings.TrimSuffix(path, ".tar.gz")

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

	// ファイルに書き込み
	targetFile := path + ".tar.gz"
	fmt.Printf("targetFile: %s\n", targetFile)

	// ディレクトリを作成
	targetDir := filepath.Dir(targetFile)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
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

	return s.WriteFile(ctx, targetFile, finalData)
}

// WriteFile はファイルをローカルファイルシステムに保存する
func (s *LocalStorage) WriteFile(ctx context.Context, key string, data []byte) error {
	// ディレクトリを作成
	dir := filepath.Dir(key)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return os.WriteFile(key, data, 0644)
}

// GetBackupData はバックアップデータを返す
func (s *LocalStorage) GetBackupData() *BackupData {
	return s.backupData
}

// SetEncryptor は暗号化処理を設定する
func (s *LocalStorage) SetEncryptor(encryptor Encryptor) {
	s.encryptor = encryptor
}
