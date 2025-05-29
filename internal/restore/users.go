package restore

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/takaishi/acb/internal/aws"
	"github.com/takaishi/acb/internal/storage"
	pkgtypes "github.com/takaishi/acb/pkg/types"
)

// Users はユーザー情報の復元を管理する
type Users struct {
	cognito *aws.CognitoClient
	storage storage.Storage
}

// NewUsers は新しいUsers構造体を作成する
func NewUsers(cognito *aws.CognitoClient, storage storage.Storage) *Users {
	return &Users{
		cognito: cognito,
		storage: storage,
	}
}

// RestoreUsers はバックアップからユーザー情報を復元する
func (u *Users) RestoreUsers(ctx context.Context, metadata *pkgtypes.BackupMetadata) error {
	// ユーザー情報ファイルのパスを構築
	usersPath := filepath.Join(metadata.UserPoolID, "users.json")

	// ユーザー情報を読み込む
	userData, err := u.storage.ReadFile(ctx, usersPath)
	if err != nil {
		return fmt.Errorf("failed to read users data: %w", err)
	}

	var usersBackup pkgtypes.UsersBackup
	if err := json.Unmarshal(userData, &usersBackup); err != nil {
		return fmt.Errorf("failed to unmarshal users data: %w", err)
	}

	// ユーザーを一括で復元
	for _, user := range usersBackup.Users {
		if err := u.restoreUser(ctx, metadata.UserPoolID, &user); err != nil {
			fmt.Printf("Warning: failed to restore user %s: %v\n", user.Username, err)
			continue
		}
	}

	return nil
}

// restoreUser は単一のユーザーを復元する
func (u *Users) restoreUser(ctx context.Context, userPoolID string, user *pkgtypes.UserInfo) error {
	// ユーザー属性を変換
	var userAttrs []types.AttributeType
	for _, attr := range user.Attributes {
		name := attr["Name"].(string)
		value := attr["Value"].(string)
		userAttrs = append(userAttrs, types.AttributeType{
			Name:  &name,
			Value: &value,
		})
	}

	// ユーザーを作成
	input := &cognitoidentityprovider.AdminCreateUserInput{
		UserPoolId:             &userPoolID,
		Username:               &user.Username,
		UserAttributes:         userAttrs,
		MessageAction:          types.MessageActionTypeSuppress,
		DesiredDeliveryMediums: []types.DeliveryMediumType{types.DeliveryMediumTypeEmail},
	}

	_, err := u.cognito.CreateUser(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	// グループメンバーシップを復元
	for _, groupName := range user.Groups {
		addToGroupInput := &cognitoidentityprovider.AdminAddUserToGroupInput{
			UserPoolId: &userPoolID,
			Username:   &user.Username,
			GroupName:  &groupName,
		}

		if _, err := u.cognito.AddUserToGroup(ctx, addToGroupInput); err != nil {
			fmt.Printf("Warning: failed to add user %s to group %s: %v\n", user.Username, groupName, err)
		}
	}

	return nil
}
