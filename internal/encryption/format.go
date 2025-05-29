package encryption

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// SerializeEncryptedData は暗号化データをバイナリ形式にシリアライズする
// フォーマット: | 4 bytes | 4 bytes | N bytes | M bytes | 残り |
//
//	| IV長    | 暗号化DK長 | IV     | 暗号化DK | 暗号化データ |
func SerializeEncryptedData(encryptedData *EncryptedData) ([]byte, error) {
	var buf bytes.Buffer

	// 暗号化データキー長を書き込み
	encKeyLen := uint32(len(encryptedData.EncryptedDataKey))
	if err := binary.Write(&buf, binary.LittleEndian, encKeyLen); err != nil {
		return nil, fmt.Errorf("failed to write encrypted data key length: %w", err)
	}

	// 暗号化データ長を書き込み
	dataLen := uint32(len(encryptedData.EncryptedData))
	if err := binary.Write(&buf, binary.LittleEndian, dataLen); err != nil {
		return nil, fmt.Errorf("failed to write encrypted data length: %w", err)
	}

	// 暗号化データキーを書き込み
	if _, err := buf.Write(encryptedData.EncryptedDataKey); err != nil {
		return nil, fmt.Errorf("failed to write encrypted data key: %w", err)
	}

	// 暗号化データを書き込み
	if _, err := buf.Write(encryptedData.EncryptedData); err != nil {
		return nil, fmt.Errorf("failed to write encrypted data: %w", err)
	}

	return buf.Bytes(), nil
}

// DeserializeEncryptedData はバイナリ形式から暗号化データをデシリアライズする
func DeserializeEncryptedData(data []byte) (*EncryptedData, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("data is too short: %d bytes", len(data))
	}

	buf := bytes.NewReader(data)

	// 暗号化データキー長を読み込み
	var encKeyLen uint32
	if err := binary.Read(buf, binary.LittleEndian, &encKeyLen); err != nil {
		return nil, fmt.Errorf("failed to read encrypted data key length: %w", err)
	}

	// 暗号化データ長を読み込み
	var dataLen uint32
	if err := binary.Read(buf, binary.LittleEndian, &dataLen); err != nil {
		return nil, fmt.Errorf("failed to read encrypted data length: %w", err)
	}

	// 残りデータ長の検証
	remainingData := len(data) - 8
	if remainingData < int(encKeyLen+dataLen) {
		return nil, fmt.Errorf("incomplete data: expected %d bytes, got %d bytes", encKeyLen+dataLen, remainingData)
	}

	// 暗号化データキーを読み込み
	encryptedDataKey := make([]byte, encKeyLen)
	if _, err := buf.Read(encryptedDataKey); err != nil {
		return nil, fmt.Errorf("failed to read encrypted data key: %w", err)
	}

	// 暗号化データを読み込み
	encryptedData := make([]byte, dataLen)
	if _, err := buf.Read(encryptedData); err != nil {
		return nil, fmt.Errorf("failed to read encrypted data: %w", err)
	}

	return &EncryptedData{
		EncryptedDataKey: encryptedDataKey,
		EncryptedData:    encryptedData,
	}, nil
}
