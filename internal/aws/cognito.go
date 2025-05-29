package aws

import (
	"context"
	"fmt"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/config"
	cognito "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

// CognitoClient はCognito操作のためのクライアントを表す
type CognitoClient struct {
	client *cognito.Client
}

// NewCognitoClient は新しいCognitoClientを作成する
func NewCognitoClient(ctx context.Context) (*CognitoClient, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	return &CognitoClient{
		client: cognito.NewFromConfig(cfg),
	}, nil
}

// ListUserPools は指定されたパターンに一致するユーザープールの一覧を取得する
func (c *CognitoClient) ListUserPools(ctx context.Context, pattern string) ([]types.UserPoolDescriptionType, error) {
	var userPools []types.UserPoolDescriptionType
	var nextToken *string

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	var maxResults int32 = 60
	for {
		input := &cognito.ListUserPoolsInput{
			MaxResults: &maxResults,
			NextToken:  nextToken,
		}

		output, err := c.client.ListUserPools(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to get user pool list: %w", err)
		}

		// パターンに一致するユーザープールをフィルタリング
		for _, pool := range output.UserPools {
			if regex.MatchString(*pool.Name) {
				userPools = append(userPools, pool)
			}
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return userPools, nil
}

// GetUserPoolConfiguration はユーザープールの設定を取得する
func (c *CognitoClient) GetUserPoolConfiguration(ctx context.Context, userPoolID string) (*cognito.DescribeUserPoolOutput, error) {
	input := &cognito.DescribeUserPoolInput{
		UserPoolId: &userPoolID,
	}

	output, err := c.client.DescribeUserPool(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get user pool configuration: %w", err)
	}

	return output, nil
}

// ListUsers はユーザープール内のすべてのユーザーを取得する
func (c *CognitoClient) ListUsers(ctx context.Context, userPoolID string) ([]types.UserType, error) {
	var users []types.UserType
	var nextToken *string

	for {
		input := &cognito.ListUsersInput{
			UserPoolId:      &userPoolID,
			PaginationToken: nextToken,
		}

		output, err := c.client.ListUsers(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to get user list: %w", err)
		}

		users = append(users, output.Users...)

		if output.PaginationToken == nil {
			break
		}
		nextToken = output.PaginationToken
	}

	return users, nil
}

// ListUserGroups はユーザーが所属するグループの一覧を取得する
func (c *CognitoClient) ListUserGroups(ctx context.Context, userPoolID, username string) ([]string, error) {
	var groups []string
	var nextToken *string

	for {
		input := &cognito.AdminListGroupsForUserInput{
			UserPoolId: &userPoolID,
			Username:   &username,
			NextToken:  nextToken,
		}

		output, err := c.client.AdminListGroupsForUser(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to get user group list: %w", err)
		}

		for _, group := range output.Groups {
			groups = append(groups, *group.GroupName)
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return groups, nil
}

// CreateUser は新しいユーザーを作成する
func (c *CognitoClient) CreateUser(ctx context.Context, input *cognito.AdminCreateUserInput) (*cognito.AdminCreateUserOutput, error) {
	output, err := c.client.AdminCreateUser(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return output, nil
}

// AddUserToGroup はユーザーをグループに追加する
func (c *CognitoClient) AddUserToGroup(ctx context.Context, input *cognito.AdminAddUserToGroupInput) (*cognito.AdminAddUserToGroupOutput, error) {
	output, err := c.client.AdminAddUserToGroup(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to add user to group: %w", err)
	}
	return output, nil
}

// CreateUserPool は新しいユーザープールを作成する
func (c *CognitoClient) CreateUserPool(ctx context.Context, input *cognito.CreateUserPoolInput) (*cognito.CreateUserPoolOutput, error) {
	output, err := c.client.CreateUserPool(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create user pool: %w", err)
	}
	return output, nil
}

// ToPoolPolicies はマップからCognitoのポリシー設定に変換する
func ToPoolPolicies(policies map[string]interface{}) *types.UserPoolPolicyType {
	if policies == nil {
		return nil
	}

	return &types.UserPoolPolicyType{
		PasswordPolicy: toPasswordPolicy(policies["password_policy"].(map[string]interface{})),
	}
}

// toPasswordPolicy はマップからパスワードポリシーに変換する
func toPasswordPolicy(policy map[string]interface{}) *types.PasswordPolicyType {
	if policy == nil {
		return nil
	}

	minimumLength := int32(policy["minimum_length"].(float64))
	requireUppercase := policy["require_uppercase"].(bool)
	requireLowercase := policy["require_lowercase"].(bool)
	requireNumbers := policy["require_numbers"].(bool)
	requireSymbols := policy["require_symbols"].(bool)

	return &types.PasswordPolicyType{
		MinimumLength:    &minimumLength,
		RequireUppercase: requireUppercase,
		RequireLowercase: requireLowercase,
		RequireNumbers:   requireNumbers,
		RequireSymbols:   requireSymbols,
	}
}

// ToMFAConfig はマップからMFA設定に変換する
func ToMFAConfig(config map[string]interface{}) types.UserPoolMfaType {
	if config == nil {
		return types.UserPoolMfaTypeOff
	}

	mfaType := config["mfa_type"].(string)
	switch mfaType {
	case "OFF":
		return types.UserPoolMfaTypeOff
	case "ON":
		return types.UserPoolMfaTypeOn
	case "OPTIONAL":
		return types.UserPoolMfaTypeOptional
	default:
		return types.UserPoolMfaTypeOff
	}
}

// ToSchemaAttributes はカスタム属性の配列からスキーマ属性に変換する
func ToSchemaAttributes(attributes []interface{}) []types.SchemaAttributeType {
	if attributes == nil {
		return nil
	}

	var schemaAttrs []types.SchemaAttributeType
	for _, attr := range attributes {
		attrMap := attr.(map[string]interface{})
		name := attrMap["name"].(string)
		attrType := attrMap["type"].(string)
		mutable := attrMap["mutable"].(bool)
		required := attrMap["required"].(bool)

		schemaAttr := types.SchemaAttributeType{
			Name:              &name,
			AttributeDataType: types.AttributeDataType(attrType),
			Mutable:           &mutable,
			Required:          &required,
		}
		schemaAttrs = append(schemaAttrs, schemaAttr)
	}

	return schemaAttrs
}

// ToLambdaConfig はマップからLambda設定に変換する
func ToLambdaConfig(config map[string]interface{}) *types.LambdaConfigType {
	if config == nil {
		return nil
	}

	lambdaConfig := &types.LambdaConfigType{}

	if preSignUp, ok := config["pre_sign_up"].(string); ok {
		lambdaConfig.PreSignUp = &preSignUp
	}
	if postConfirmation, ok := config["post_confirmation"].(string); ok {
		lambdaConfig.PostConfirmation = &postConfirmation
	}
	if preAuthentication, ok := config["pre_authentication"].(string); ok {
		lambdaConfig.PreAuthentication = &preAuthentication
	}
	if postAuthentication, ok := config["post_authentication"].(string); ok {
		lambdaConfig.PostAuthentication = &postAuthentication
	}

	return lambdaConfig
}
