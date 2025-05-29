package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/takaishi/acb/internal/config"
	"github.com/takaishi/acb/internal/encryption"
	"github.com/takaishi/acb/internal/storage"
)

func Decrypt(cli *CLI) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Validate AWS credentials
	if err := config.ValidateAWSCredentials(ctx); err != nil {
		return err
	}

	inputInfo, err := parseStorageURI(cli.Decrypt.Input)
	if err != nil {
		return fmt.Errorf("failed to parse input path: %w", err)
	}
	var inputStore storage.Storage
	switch inputInfo.storageType {
	case "s3":
		inputStore, err = storage.NewS3Storage(ctx, inputInfo.bucket)
		if err != nil {
			return fmt.Errorf("failed to initialize S3 storage: %w", err)
		}
	case "file":
		inputStore, err = storage.NewLocalStorage()
		if err != nil {
			return fmt.Errorf("failed to initialize local storage: %w", err)
		}
	}

	outputInfo, err := parseStorageURI(cli.Decrypt.Output)
	if err != nil {
		return fmt.Errorf("failed to parse output path: %w", err)
	}
	var outputStore storage.Storage
	switch outputInfo.storageType {
	case "s3":
		outputStore, err = storage.NewS3Storage(ctx, outputInfo.bucket)
		if err != nil {
			return fmt.Errorf("failed to initialize S3 storage: %w", err)
		}
	case "file":
		outputStore, err = storage.NewLocalStorage()
		if err != nil {
			return fmt.Errorf("failed to initialize local storage: %w", err)
		}
	}

	dataKeyInfo, err := parseStorageURI(cli.Decrypt.DataKeyPath)
	if err != nil {
		return fmt.Errorf("failed to parse data key path: %w", err)
	}
	fmt.Printf("dataKeyInfo: %+v\n", dataKeyInfo)

	// Read data key file
	dataKey, err := readDataKey(dataKeyInfo)
	if err != nil {
		return fmt.Errorf("failed to read data key file: %w", err)
	}

	// Initialize KMSEncryptor
	encryptor, err := encryption.NewKMSEncryptor(ctx, cli.Decrypt.KMSKeyID, cli.Decrypt.KMSRegion)
	if err != nil {
		return fmt.Errorf("failed to initialize KMSEncryptor: %w", err)
	}
	encryptor.SetDataKey(dataKey)

	// Read encrypted file
	encryptedData, err := inputStore.ReadFile(ctx, inputInfo.path)
	if err != nil {
		return fmt.Errorf("failed to read encrypted file: %w", err)
	}

	// Deserialize encrypted data
	encryptedDataObj, err := encryption.DeserializeEncryptedData(encryptedData)
	if err != nil {
		return fmt.Errorf("failed to deserialize encrypted data: %w", err)
	}

	// Decrypt
	decryptedData, err := encryptor.Decrypt(ctx, encryptedDataObj)
	if err != nil {
		return fmt.Errorf("failed to decrypt file: %w", err)
	}

	// Save decrypted data
	if err := outputStore.WriteFile(ctx, outputInfo.path, decryptedData); err != nil {
		return fmt.Errorf("failed to save decrypted file: %w", err)
	}

	fmt.Printf("Decryption completed: %s\n", cli.Decrypt.Output)
	return nil
}
