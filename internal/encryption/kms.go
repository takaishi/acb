package encryption

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
)

// EncryptedData は暗号化されたデータを表す
type EncryptedData struct {
	EncryptedDataKey []byte
	EncryptedData    []byte
}

// KMSEncryptor はKMSを使用した暗号化を担当する構造体
type KMSEncryptor struct {
	kmsClient *kms.Client
	keyID     string
	dataKey   []byte
}

// NewKMSEncryptor は新しいKMSEncryptorインスタンスを作成する
func NewKMSEncryptor(ctx context.Context, keyID string, region string) (*KMSEncryptor, error) {
	fmt.Printf("region: %s\n", region)
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("AWS設定の読み込みに失敗しました: %w", err)
	}
	client := kms.NewFromConfig(cfg, func(o *kms.Options) {
		o.Region = region
	})

	return &KMSEncryptor{
		kmsClient: client,
		keyID:     keyID,
	}, nil
}

// SetDataKey はデータキーを設定する
func (e *KMSEncryptor) SetDataKey(dataKey []byte) {
	e.dataKey = dataKey
}

// Encrypt はデータを暗号化する
func (e *KMSEncryptor) Encrypt(ctx context.Context, data []byte) (*EncryptedData, error) {
	// データキーが設定されていない場合は新しく生成
	if e.dataKey == nil {
		dataKeyInfo, err := e.GenerateDataKey(ctx)
		if err != nil {
			return nil, fmt.Errorf("データキーの生成に失敗しました: %w", err)
		}
		e.dataKey = dataKeyInfo.Plaintext
	}
	decryptOutput, err := e.kmsClient.Decrypt(ctx, &kms.DecryptInput{
		KeyId:          aws.String(e.keyID),
		CiphertextBlob: e.dataKey,
	})
	if err != nil {
		return nil, fmt.Errorf("データキーの復号化に失敗しました: %w", err)
	}
	e.dataKey = decryptOutput.Plaintext

	// AES-GCM暗号化を使用
	block, err := aes.NewCipher(e.dataKey)
	if err != nil {
		return nil, fmt.Errorf("AES暗号化の初期化に失敗しました: %w", err)
	}

	// GCMモードで暗号化
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("GCMモードの初期化に失敗しました: %w", err)
	}

	// ランダムなIVを生成
	iv := make([]byte, aesGCM.NonceSize())
	if _, err := rand.Read(iv); err != nil {
		return nil, fmt.Errorf("IVの生成に失敗しました: %w", err)
	}

	// 暗号化
	ciphertext := aesGCM.Seal(iv, iv, data, nil)

	return &EncryptedData{
		EncryptedDataKey: e.dataKey,
		EncryptedData:    ciphertext,
	}, nil
}

// Decrypt はデータを復号化する
func (e *KMSEncryptor) Decrypt(ctx context.Context, encryptedData *EncryptedData) ([]byte, error) {
	// KMSでデータキーを復号化
	decryptOutput, err := e.kmsClient.Decrypt(ctx, &kms.DecryptInput{
		KeyId:          aws.String(e.keyID),
		CiphertextBlob: e.dataKey,
	})
	if err != nil {
		return nil, fmt.Errorf("データキーの復号化に失敗しました: %w", err)
	}
	e.dataKey = decryptOutput.Plaintext

	// AES-GCM暗号化を使用
	block, err := aes.NewCipher(e.dataKey)
	if err != nil {
		return nil, fmt.Errorf("AES暗号化の初期化に失敗しました: %w", err)
	}

	// GCMモードで復号化
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("GCMモードの初期化に失敗しました: %w", err)
	}

	// IVを抽出（先頭12バイト）
	if len(encryptedData.EncryptedData) < aesGCM.NonceSize() {
		return nil, fmt.Errorf("暗号化データが不正です")
	}
	iv := encryptedData.EncryptedData[:aesGCM.NonceSize()]
	ciphertext := encryptedData.EncryptedData[aesGCM.NonceSize():]

	// 復号化
	plaintext, err := aesGCM.Open(nil, iv, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("データの復号化に失敗しました: %w", err)
	}

	return plaintext, nil
}

// GenerateDataKey はKMSからデータキーを生成する
func (e *KMSEncryptor) GenerateDataKey(ctx context.Context) (*kms.GenerateDataKeyOutput, error) {
	output, err := e.kmsClient.GenerateDataKey(ctx, &kms.GenerateDataKeyInput{
		KeyId:   aws.String(e.keyID),
		KeySpec: "AES_256",
	})
	if err != nil {
		return nil, fmt.Errorf("データキーの生成に失敗しました: %w", err)
	}

	return output, nil
}
