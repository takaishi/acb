package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/takaishi/acb/internal/aws"
	"github.com/takaishi/acb/internal/backup"
	"github.com/takaishi/acb/internal/config"
	"github.com/takaishi/acb/internal/encryption"
	"github.com/takaishi/acb/internal/storage"
)

func Backup(cli *CLI) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Parse URI
	info, err := parseStorageURI(cli.Backup.URI)
	if err != nil {
		return err
	}

	// Validate AWS credentials
	if info.storageType == "s3" {
		if err := config.ValidateAWSCredentials(ctx); err != nil {
			return err
		}
	}

	// Initialize Cognito client
	cognitoClient, err := aws.NewCognitoClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize Cognito client: %w", err)
	}

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Update KMS settings
	if cli.Backup.KMSKeyID != "" {
		cfg.KMS.Enabled = true
		cfg.KMS.KeyID = cli.Backup.KMSKeyID
	}
	if cli.Backup.DataKeyPath != "" {
		cfg.KMS.DataKeyPath = cli.Backup.DataKeyPath
	}

	dataKeyInfo, err := parseStorageURI(cli.Backup.DataKeyPath)
	if err != nil {
		return fmt.Errorf("failed to parse data key path: %w", err)
	}
	fmt.Printf("dataKeyInfo: %+v\n", dataKeyInfo)

	// Initialize storage
	var store storage.Storage
	switch info.storageType {
	case "s3":
		store, err = storage.NewS3Storage(ctx, info.bucket)
		if err != nil {
			return fmt.Errorf("failed to initialize S3 storage: %w", err)
		}
	case "file":
		store, err = storage.NewLocalStorage()
		if err != nil {
			return fmt.Errorf("failed to initialize local storage: %w", err)
		}
	}

	// Configure KMS encryption
	if cfg.KMS.Enabled {
		// Read data key file
		dataKey, err := readDataKey(dataKeyInfo)
		if err != nil {
			return fmt.Errorf("failed to read data key file: %w", err)
		}

		// Decode plaintext data key
		// dataKey, err := base64.StdEncoding.DecodeString(string(data))
		// if err != nil {
		// 	return fmt.Errorf("failed to decode plaintext data key: %w", err)
		// }

		// Initialize encryption handler
		encryptor, err := encryption.NewKMSEncryptor(ctx, cfg.KMS.KeyID, cli.Backup.KMSRegion)
		if err != nil {
			return fmt.Errorf("failed to initialize KMS encryption: %w", err)
		}

		encryptor.SetDataKey(dataKey)
		store.SetEncryptor(encryptor)
		fmt.Printf("KMS encryption enabled (KeyID: %s)\n", cfg.KMS.KeyID)
	}

	// Execute backup
	backupper := backup.NewPoolBackupper(cognitoClient, store)
	if err := backupper.BackupPools(ctx, cli.Backup.Pattern, info.path); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	fmt.Println("Backup completed")

	// Compress backup directory
	if err := store.CreateTarGz(ctx, info.path); err != nil {
		return fmt.Errorf("failed to compress backup: %w", err)
	}

	fmt.Println("Backup compression completed")
	return nil
}
