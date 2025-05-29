# AWS Cognito Backuper

AWS Cognito Backuper is a CLI tool that comprehensively backs up AWS Cognito user pools and their associated user information, allowing restoration to different regions or accounts as needed.

## Features

- Bulk backup of multiple Cognito user pools
- Filtering user pools using regular expressions
- Individual backup file generation for each user pool
- Backup storage to S3 or local file system
- KMS encryption support for secure backups
- Data key validation for AES-256 encryption

## Installation

```bash
go install github.com/takaishi/acb/cmd/acb@latest
```

Or build from source:

```bash
git clone https://github.com/takaishi/acb.git
cd acb
make
```

## Usage

### AWS Credentials Setup

Set up AWS credentials using one of the following methods:

- AWS CLI configuration (`aws configure`)
- Environment variables
  ```bash
  export AWS_ACCESS_KEY_ID=your_access_key
  export AWS_SECRET_ACCESS_KEY=your_secret_key
  export AWS_REGION=ap-northeast-1
  ```

### Available Commands

```bash
# Show help
acb --help

# Show command help
acb backup --help
acb list --help
```

### List User Pools

```bash
# List all user pools
acb list

# Filter with regular expression
acb list --pattern="foo-.*"
```

### Backup

S3 backup:

```bash
# Backup all user pools to S3
acb backup --uri="s3://your-backup-bucket/backups"

# Backup specific user pools to S3
acb backup --pattern="foo-.*" --uri="s3://your-backup-bucket/backups"

# Backup with KMS encryption
acb backup --uri="s3://your-backup-bucket/backups" --kms-key-id="alias/my-key" --data-key-path="file:///path/to/datakey.json" --kms-region="ap-northeast-1"
```

Local backup:

```bash
# Backup all user pools locally
acb backup --uri="file:///path/to/backups"

# Backup specific user pools locally
acb backup --pattern="foo-.*" --uri="file:///path/to/backups"

# Backup with KMS encryption
acb backup --uri="file:///path/to/backups" --kms-key-id="alias/my-key" --data-key-path="file:///path/to/datakey.json" --kms-region="ap-northeast-1"
```

### Restore

S3 restore:

```bash
# Restore all backups
acb restore --uri="s3://your-backup-bucket/backups"

# Restore with prefix
acb restore --uri="s3://your-backup-bucket/backups" --pattern="foo-.*"

# Restore specific user pool backups
acb restore --uri="s3://your-backup-bucket/backups" --pattern="foo-.*"
```

### Generate Data Key

```bash
# Generate a new data key
acb generate-datakey --kms-key-id="alias/my-key" --kms-region="ap-northeast-1" --output="datakey.json"

# Generate a test data key
acb generate-datakey --kms-key-id="alias/my-key" --kms-region="ap-northeast-1" --test --output="test-datakey.json"
```

### Backup File Structure

S3:

```
s3://<bucket>/<prefix>/YYYY-MM-DD/<user-pool-id>/
  - metadata.json      # Backup metadata
  - pool-config.json   # Pool configuration
  - users.json         # User information
```

Local:

```
<local-path>/<prefix>/<user-pool-id>/
  - metadata.json      # Backup metadata
  - pool-config.json   # Pool configuration
  - users.json         # User information
```

## Required Permissions

The tool requires the following IAM permissions:

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

Note:

- S3 permissions are not required when using local storage
- KMS permissions are only required when using encryption
- Restore functionality only supports restoration from S3

## Development

### Build

```bash
make build
```

### Test

```bash
make test
```

### Lint

```bash
make lint
```

## License

MIT License
