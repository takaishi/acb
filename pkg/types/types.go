package types

// BackupMetadata はバックアップのメタデータを表す
type BackupMetadata struct {
	Version       string   `json:"version"`
	Timestamp     string   `json:"timestamp"`
	SourceAccount string   `json:"source_account"`
	SourceRegion  string   `json:"source_region"`
	UserPoolID    string   `json:"user_pool_id"`
	BackupFiles   []string `json:"backup_files"`
}

// PoolConfiguration はユーザープールの設定を表す
type PoolConfiguration struct {
	Policies         map[string]interface{} `json:"policies"`
	MFAConfiguration map[string]interface{} `json:"mfa_configuration"`
	CustomAttributes []interface{}          `json:"custom_attributes"`
	Triggers         map[string]interface{} `json:"triggers"`
}

// UserInfo はユーザー情報を表す
type UserInfo struct {
	Username    string                   `json:"username"`
	Attributes  []map[string]interface{} `json:"attributes"`
	Groups      []string                 `json:"groups"`
	MFASettings map[string]interface{}   `json:"mfa_settings"`
}

// UsersBackup はユーザー情報のバックアップを表す
type UsersBackup struct {
	Users []UserInfo `json:"users"`
}

// BackupOptions はバックアップ操作のオプションを表す
type BackupOptions struct {
	Pattern  string
	S3Bucket string
	S3Prefix string
}

// RestoreOptions は復元操作のオプションを表す
type RestoreOptions struct {
	SourceBucket string
	SourcePrefix string
	Pattern      string
}
