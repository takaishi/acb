package encryption

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/kms"
)

// DataKeyInfo はデータキー情報を保持する
type DataKeyInfo struct {
	PlaintextKey []byte // 平文データキー（暗号化処理後は即座に削除）
	EncryptedKey []byte // 暗号化されたデータキー（保存用）
}

// Encryptor は暗号化/復号化のインターフェース
type Encryptor interface {
	// データキーを設定する
	SetDataKey(dataKey []byte)

	// データを暗号化する
	Encrypt(ctx context.Context, data []byte) (*EncryptedData, error)

	// 暗号化されたデータを復号化する
	Decrypt(ctx context.Context, encryptedData *EncryptedData) ([]byte, error)

	// データキーを生成する
	GenerateDataKey(ctx context.Context) (*kms.GenerateDataKeyOutput, error)
}
