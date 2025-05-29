package cmd

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/takaishi/acb/internal/config"
	"github.com/takaishi/acb/internal/encryption"
)

// DataKeyResult はデータキー生成結果を表す
type DataKeyResult struct {
	KMSKeyID         string `json:"kms_key_id"`
	EncryptedDataKey string `json:"encrypted_data_key"`
	PlaintextDataKey string `json:"plaintext_data_key,omitempty"`
	KeySpec          string `json:"key_spec"`
	GeneratedAt      string `json:"generated_at"`
}

func GenerateDatakey(cli *CLI) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if cli.GenerateDatakey.Test {
		return generateTestDatakey(cli)
	}

	// Validate AWS credentials
	if err := config.ValidateAWSCredentials(ctx); err != nil {
		return err
	}

	// Initialize KMSEncryptor
	encryptor, err := encryption.NewKMSEncryptor(ctx, cli.GenerateDatakey.KMSKeyID, cli.GenerateDatakey.KMSRegion)
	if err != nil {
		return fmt.Errorf("failed to initialize KMSEncryptor: %w", err)
	}

	// Generate data key
	dataKeyInfo, err := encryptor.GenerateDataKey(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate data key: %w", err)
	}

	// Create result
	result := DataKeyResult{
		KMSKeyID:         cli.GenerateDatakey.KMSKeyID,
		EncryptedDataKey: base64.StdEncoding.EncodeToString(dataKeyInfo.CiphertextBlob),
		KeySpec:          cli.GenerateDatakey.Spec,
		GeneratedAt:      time.Now().UTC().Format(time.RFC3339),
	}

	// Output based on format
	var output string
	switch cli.GenerateDatakey.Format {
	case "json":
		result.PlaintextDataKey = base64.StdEncoding.EncodeToString(dataKeyInfo.Plaintext)
		jsonData, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to convert to JSON: %w", err)
		}
		output = string(jsonData)
	case "base64":
		output = fmt.Sprintf("# Encrypted Data Key (Base64)\n%s\n\n# Plaintext Data Key (Base64) - Handle with care\n%s\n",
			base64.StdEncoding.EncodeToString(dataKeyInfo.CiphertextBlob),
			base64.StdEncoding.EncodeToString(dataKeyInfo.Plaintext))
	}

	// Clear plaintext data key from memory
	for i := range dataKeyInfo.Plaintext {
		dataKeyInfo.Plaintext[i] = 0
	}

	// Output
	if cli.GenerateDatakey.Output != "" {
		err := os.WriteFile(cli.GenerateDatakey.Output, []byte(output), 0600)
		if err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		fmt.Printf("Data key saved to %s\n", cli.GenerateDatakey.Output)
	} else {
		fmt.Print(output)
	}

	return nil
}

func generateTestDatakey(cli *CLI) error {
	// Generate dummy data key for testing
	plaintextKey := make([]byte, 32)  // AES-256
	encryptedKey := make([]byte, 128) // Dummy encrypted key

	// Fill with random data
	if _, err := rand.Read(plaintextKey); err != nil {
		return fmt.Errorf("failed to generate test data key: %w", err)
	}
	if _, err := rand.Read(encryptedKey); err != nil {
		return fmt.Errorf("failed to generate test encrypted key: %w", err)
	}

	// Create result
	result := DataKeyResult{
		KMSKeyID:         cli.GenerateDatakey.KMSKeyID + "-TEST",
		EncryptedDataKey: base64.StdEncoding.EncodeToString(encryptedKey),
		KeySpec:          cli.GenerateDatakey.Spec,
		GeneratedAt:      time.Now().UTC().Format(time.RFC3339),
	}

	// Output based on format
	var output string
	switch cli.GenerateDatakey.Format {
	case "json":
		result.PlaintextDataKey = base64.StdEncoding.EncodeToString(plaintextKey)
		jsonData, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to convert to JSON: %w", err)
		}
		output = string(jsonData)
	case "base64":
		output = fmt.Sprintf("# Test Encrypted Data Key (Base64)\n%s\n\n# Test Plaintext Data Key (Base64)\n%s\n",
			base64.StdEncoding.EncodeToString(encryptedKey),
			base64.StdEncoding.EncodeToString(plaintextKey))
	}

	// Clear test data key
	for i := range plaintextKey {
		plaintextKey[i] = 0
	}

	// Output
	if cli.GenerateDatakey.Output != "" {
		err := os.WriteFile(cli.GenerateDatakey.Output, []byte(output), 0600)
		if err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		fmt.Printf("Test data key saved to %s\n", cli.GenerateDatakey.Output)
	} else {
		fmt.Print(output)
	}

	return nil
}
