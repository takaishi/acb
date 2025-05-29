package backup

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/takaishi/acb/internal/aws"
	"github.com/takaishi/acb/internal/storage"
	"github.com/takaishi/acb/pkg/types"
)

// PoolBackupper はユーザープールのバックアップを行う
type PoolBackupper struct {
	cognitoClient *aws.CognitoClient
	storage       storage.Storage
}

// NewPoolBackupper は新しいPoolBackupperを作成する
func NewPoolBackupper(cognitoClient *aws.CognitoClient, storage storage.Storage) *PoolBackupper {
	return &PoolBackupper{
		cognitoClient: cognitoClient,
		storage:       storage,
	}
}

// BackupPools は指定されたパターンに一致するユーザープールをバックアップする
func (b *PoolBackupper) BackupPools(ctx context.Context, pattern, prefix string) error {
	// ユーザープールの一覧を取得
	pools, err := b.cognitoClient.ListUserPools(ctx, pattern)
	if err != nil {
		return fmt.Errorf("failed to get user pool list: %w", err)
	}

	if len(pools) == 0 {
		return fmt.Errorf("no user pools found matching pattern: %s", pattern)
	}

	// 並行処理用のエラーチャネル
	errCh := make(chan error, len(pools))
	var wg sync.WaitGroup

	// 各ユーザープールを並行してバックアップ
	for _, pool := range pools {
		wg.Add(1)
		go func(pool string) {
			defer wg.Done()
			if err := b.backupSinglePool(ctx, pool, prefix); err != nil {
				errCh <- fmt.Errorf("failed to backup user pool %s: %w", pool, err)
			}
		}(*pool.Id)
	}

	// 完了を待機
	wg.Wait()
	close(errCh)

	// エラーの確認
	var errors []error
	for err := range errCh {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("%d errors occurred during backup: %v", len(errors), errors)
	}

	return nil
}

// backupSinglePool は単一のユーザープールをバックアップする
func (b *PoolBackupper) backupSinglePool(ctx context.Context, userPoolID, prefix string) error {
	// プール設定の取得
	poolConfig, err := b.cognitoClient.GetUserPoolConfiguration(ctx, userPoolID)
	if err != nil {
		return fmt.Errorf("failed to get user pool configuration: %w", err)
	}

	// ユーザー一覧の取得
	users, err := b.cognitoClient.ListUsers(ctx, userPoolID)
	if err != nil {
		return fmt.Errorf("failed to get user list: %w", err)
	}

	// ユーザー情報の詳細を取得
	var usersBackup types.UsersBackup
	for _, user := range users {
		groups, err := b.cognitoClient.ListUserGroups(ctx, userPoolID, *user.Username)
		if err != nil {
			return fmt.Errorf("failed to get user groups: %w", err)
		}

		userInfo := types.UserInfo{
			Username: *user.Username,
			Groups:   groups,
		}

		// 属性の変換
		attributes := make([]map[string]interface{}, len(user.Attributes))
		for i, attr := range user.Attributes {
			attributes[i] = map[string]interface{}{
				"Name":  *attr.Name,
				"Value": *attr.Value,
			}
		}
		userInfo.Attributes = attributes

		usersBackup.Users = append(usersBackup.Users, userInfo)
	}

	// メタデータの作成
	metadata := types.BackupMetadata{
		Version:       "1.0",
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		SourceAccount: "", // TODO: Get from session
		SourceRegion:  "", // TODO: Get from session
		UserPoolID:    userPoolID,
		BackupFiles:   []string{"pool-config.json", "users.json"},
	}

	// ストレージへの保存
	if err := b.storage.SaveJSON(ctx, prefix, userPoolID, "metadata.json", metadata); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	if err := b.storage.SaveJSON(ctx, prefix, userPoolID, "pool-config.json", poolConfig.UserPool); err != nil {
		return fmt.Errorf("failed to save pool configuration: %w", err)
	}

	if err := b.storage.SaveJSON(ctx, prefix, userPoolID, "users.json", usersBackup); err != nil {
		return fmt.Errorf("failed to save user information: %w", err)
	}

	return nil
}
