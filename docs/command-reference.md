# AWS Cognito Backuper コマンドリファレンス

## 1. インストール方法

### Go install を使用する場合

```bash
go install github.com/takaishi/acb/cmd/cognito-backup@latest
```

### ソースからビルドする場合

```bash
git clone https://github.com/flyle-io/flyle-tools.git
cd flyle-tools/aws_cognito_backuper
make
```

## 2. AWS 認証情報の設定

以下のいずれかの方法で AWS 認証情報を設定します：

### AWS CLI 設定を使用

```bash
aws configure
```

### 環境変数を使用

```bash
export AWS_ACCESS_KEY_ID=your_access_key
export AWS_SECRET_ACCESS_KEY=your_secret_key
export AWS_REGION=ap-northeast-1
```

## 3. コマンド一覧

### 3.1 ヘルプの表示

```bash
# 全体的なヘルプ
cognito-backup --help

# コマンド別のヘルプ
cognito-backup backup --help
cognito-backup list --help
```

### 3.2 ユーザープール一覧の表示

```bash
# 全ユーザープールの表示
cognito-backup list

# 正規表現によるフィルタリング
cognito-backup list --pattern="prod-.*"
```

### 3.3 バックアップ

#### S3 へのバックアップ

```bash
# 全ユーザープールのバックアップ
cognito-backup backup --uri="s3://your-backup-bucket/backups"

# 特定のユーザープールのバックアップ
cognito-backup backup --pattern="prod-.*" --uri="s3://your-backup-bucket/backups"

# KMS暗号化を使用したバックアップ
cognito-backup backup --uri="s3://your-backup-bucket/backups" --kms-key-id="alias/my-key" --data-key-path="file:///path/to/datakey.json"
```

#### ローカルへのバックアップ

```bash
# 全ユーザープールのバックアップ
cognito-backup backup --uri="file:///path/to/backups"

# 特定のユーザープールのバックアップ
cognito-backup backup --pattern="prod-.*" --uri="file:///path/to/backups"

# KMS暗号化を使用したバックアップ
cognito-backup backup --uri="file:///path/to/backups" --kms-key-id="alias/my-key" --data-key-path="file:///path/to/datakey.json"
```

### 3.4 復元

#### S3 からの復元

```bash
# 全バックアップの復元
cognito-backup restore --uri="s3://your-backup-bucket/backups"

# プレフィックスを指定した復元
cognito-backup restore --uri="s3://your-backup-bucket/backups" --pattern="prod-.*"

# 特定のユーザープールバックアップの復元
cognito-backup restore --uri="s3://your-backup-bucket/backups" --pattern="prod-.*"
```

### 3.5 データキーの生成

```bash
# 新しいデータキーの生成
cognito-backup generate-datakey --kms-key-id="alias/my-key" --output="datakey.json"

# テスト用データキーの生成
cognito-backup generate-datakey --kms-key-id="alias/my-key" --test --output="test-datakey.json"
```

## 4. バックアップファイル構造

### S3 の場合

```
s3://<bucket>/<prefix>/YYYY-MM-DD/<user-pool-id>/
  - metadata.json      # バックアップメタデータ
  - pool-config.json   # プール設定
  - users.json         # ユーザー情報
```

### ローカルの場合

```
<local-path>/<prefix>/<user-pool-id>/
  - metadata.json      # バックアップメタデータ
  - pool-config.json   # プール設定
  - users.json         # ユーザー情報
```

## 5. 必要な IAM 権限

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "cognito-idp:ListUserPools",
        "cognito-idp:DescribeUserPool",
        "cognito-idp:ListUsers",
        "cognito-idp:AdminListGroupsForUser",
        "cognito-idp:CreateUserPool",
        "cognito-idp:AdminCreateUser",
        "cognito-idp:AdminAddUserToGroup"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": ["s3:PutObject", "s3:GetObject", "s3:ListBucket"],
      "Resource": [
        "arn:aws:s3:::your-backup-bucket",
        "arn:aws:s3:::your-backup-bucket/*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": ["kms:GenerateDataKey", "kms:Decrypt", "kms:DescribeKey"],
      "Resource": "arn:aws:kms:region:account:key/key-id"
    }
  ]
}
```

注意事項：

- S3 権限はローカルストレージを使用する場合は不要です
- KMS 権限は暗号化を使用する場合のみ必要です
- 復元機能は S3 からの復元のみをサポートしています
