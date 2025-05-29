package storage

import (
	"context"

	"github.com/takaishi/acb/internal/encryption"
)

// BackupFile はバックアップファイルを表す
type BackupFile struct {
	Name    string
	Content []byte
}

// BackupData はバックアップデータを保持する
type BackupData struct {
	Files map[string][]BackupFile // userPoolID -> files
}

// NewBackupData は新しいBackupDataを作成する
func NewBackupData() *BackupData {
	return &BackupData{
		Files: make(map[string][]BackupFile),
	}
}

// Storage はバックアップの保存先を表すインターフェース
type Storage interface {
	SaveJSON(ctx context.Context, prefix, userPoolID string, filename string, data interface{}) error
	WriteFile(ctx context.Context, key string, data []byte) error
	ReadFile(ctx context.Context, key string) ([]byte, error)
	CreateTarGz(ctx context.Context, prefix string) error
	GetBackupData() *BackupData
	// 暗号化関連の新規追加
	SetEncryptor(encryptor Encryptor)
}

// Encryptor は暗号化/復号化のインターフェース
type Encryptor = encryption.Encryptor

// isEncryptedData はデータが暗号化されているかを判定する
// 暗号化されたデータは最初の8バイトが長さ情報（IV長 + 暗号化DK長）になっている
func isEncryptedData(data []byte) bool {
	// 最低限のヘッダーサイズをチェック
	if len(data) < 8 {
		return false
	}

	// 最初の4バイトをIV長として読み込み
	ivLen := uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
	// 次の4バイトを暗号化DK長として読み込み
	encKeyLen := uint32(data[4]) | uint32(data[5])<<8 | uint32(data[6])<<16 | uint32(data[7])<<24

	// 妥当な範囲内かチェック（IV: 12バイト、暗号化DK: 100-1000バイト程度）
	if ivLen == 12 && encKeyLen >= 50 && encKeyLen <= 2000 {
		// データ全体のサイズが期待通りかチェック
		expectedMinSize := 8 + int(ivLen) + int(encKeyLen) + 16 // 最低限の暗号化データサイズ
		return len(data) >= expectedMinSize
	}

	return false
}
