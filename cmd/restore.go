package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path"
	"syscall"

	"github.com/takaishi/acb/internal/aws"
	"github.com/takaishi/acb/internal/config"
	"github.com/takaishi/acb/internal/encryption"
	"github.com/takaishi/acb/internal/restore"
	"github.com/takaishi/acb/internal/storage"
	"github.com/takaishi/acb/pkg/types"
)

func Restore(cli *CLI) error {
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
	info, err := parseStorageURI(cli.Restore.URI)
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

	// Initialize storage
	var store storage.Storage
	var backups []string

	switch info.storageType {
	case "s3":
		// Initialize S3 client
		s3Client, err := aws.NewS3Client(ctx)
		if err != nil {
			return fmt.Errorf("failed to initialize S3 client: %w", err)
		}

		// Get backup list
		backups, err = s3Client.ListBackups(ctx, info.bucket, info.path, cli.Restore.Pattern)
		if err != nil {
			return fmt.Errorf("failed to list backups: %w", err)
		}

		store, err = storage.NewS3Storage(ctx, info.bucket)
		if err != nil {
			return fmt.Errorf("failed to initialize S3 storage: %w", err)
		}

	case "file":
		store, err = storage.NewLocalStorage()
		if err != nil {
			return fmt.Errorf("failed to initialize local storage: %w", err)
		}
		// For local storage, specify a single backup file
		backups = []string{info.path}
	}

	// Configure KMS encryption
	if cfg.KMS.Enabled {
		encryptor, err := encryption.NewKMSEncryptor(ctx, cfg.KMS.KeyID, cli.Restore.KMSRegion)
		if err != nil {
			return fmt.Errorf("failed to initialize KMS encryption: %w", err)
		}
		store.SetEncryptor(encryptor)
		fmt.Printf("KMS encryption enabled (KeyID: %s)\n", cfg.KMS.KeyID)
	}

	if len(backups) == 0 {
		return fmt.Errorf("no backups found matching the specified pattern")
	}

	fmt.Printf("Backups to restore: %d\n", len(backups))

	// Initialize pool restorer
	poolRestorer := restore.NewPool(cognitoClient, store)
	userRestorer := restore.NewUsers(cognitoClient, store)

	// Restore each backup
	for _, backupPath := range backups {
		// Read metadata
		metadataPath := path.Join(backupPath, "metadata.json")
		metadataData, err := store.ReadFile(ctx, metadataPath)
		if err != nil {
			fmt.Printf("Warning: Failed to read metadata (%s): %v\n", backupPath, err)
			continue
		}

		var metadata types.BackupMetadata
		if err := json.Unmarshal(metadataData, &metadata); err != nil {
			fmt.Printf("Warning: Failed to parse metadata (%s): %v\n", backupPath, err)
			continue
		}

		fmt.Printf("Starting restoration of user pool %s...\n", metadata.UserPoolID)

		// Restore user pool
		if err := poolRestorer.RestorePool(ctx, &metadata); err != nil {
			fmt.Printf("Warning: Failed to restore user pool (%s): %v\n", metadata.UserPoolID, err)
			continue
		}

		// Restore user information
		if err := userRestorer.RestoreUsers(ctx, &metadata); err != nil {
			fmt.Printf("Warning: Failed to restore user information (%s): %v\n", metadata.UserPoolID, err)
			continue
		}

		fmt.Printf("Restoration of user pool %s completed\n", metadata.UserPoolID)
	}

	fmt.Println("Restoration completed")
	return nil
}
