package restore

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/takaishi/acb/internal/aws"
	"github.com/takaishi/acb/internal/storage"
	"github.com/takaishi/acb/pkg/types"
)

// Pool はユーザープールの復元を管理する
type Pool struct {
	cognito *aws.CognitoClient
	storage storage.Storage
}

// NewPool は新しいPool構造体を作成する
func NewPool(cognito *aws.CognitoClient, storage storage.Storage) *Pool {
	return &Pool{
		cognito: cognito,
		storage: storage,
	}
}

// RestorePool はバックアップからユーザープールを復元する
func (p *Pool) RestorePool(ctx context.Context, metadata *types.BackupMetadata) error {
	// メタデータからプール設定ファイルのパスを構築
	configPath := filepath.Join(metadata.UserPoolID, "pool-config.json")

	// プール設定を読み込む
	configData, err := p.storage.ReadFile(ctx, configPath)
	if err != nil {
		return fmt.Errorf("failed to read pool config: %w", err)
	}

	var poolConfig types.PoolConfiguration
	if err := json.Unmarshal(configData, &poolConfig); err != nil {
		return fmt.Errorf("failed to unmarshal pool config: %w", err)
	}

	// ユーザープールを作成
	input := &cognitoidentityprovider.CreateUserPoolInput{
		PoolName: &metadata.UserPoolID,
		// 基本設定を適用
		Policies:         aws.ToPoolPolicies(poolConfig.Policies),
		MfaConfiguration: aws.ToMFAConfig(poolConfig.MFAConfiguration),
		// カスタム属性を適用
		Schema: aws.ToSchemaAttributes(poolConfig.CustomAttributes),
		// Lambda トリガーを適用
		LambdaConfig: aws.ToLambdaConfig(poolConfig.Triggers),
	}

	// ユーザープールを作成
	output, err := p.cognito.CreateUserPool(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create user pool: %w", err)
	}

	fmt.Printf("Successfully restored user pool: %s\n", *output.UserPool.Id)
	return nil
}
