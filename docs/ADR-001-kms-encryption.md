# ADR-001: KMS暗号化による Cognito バックアップデータの保護

## ステータス
提案中

## コンテキスト
aws_cognito_backuper は現在、ユーザープールの設定とユーザーデータをS3またはローカルファイルシステムにJSON形式で保存している。これらのバックアップデータには機密情報（ユーザー属性、パスワードハッシュなど）が含まれるため、暗号化によるデータ保護が必要である。

現在の実装では以下の課題がある：
- バックアップデータが平文で保存されている
- 機密情報の漏洩リスクが存在する
- コンプライアンス要件を満たしていない可能性がある

## 決定
AWS KMS（Key Management Service）を使用してバックアップデータを暗号化する。

### 暗号化アーキテクチャ

1. **暗号化対象**
   - tar.gz圧縮後のバックアップファイル全体
   - 個別のJSONファイルではなく、アーカイブレベルで暗号化

2. **エンベロープ暗号化パターン**
   - AWS KMS データキーを使用したエンベロープ暗号化
   - カスタマー管理キー（CMK）でデータキーを保護
   - 実際のデータはデータキーで暗号化（AES-256-GCM）

3. **KMS キー管理**
   - カスタマー管理キー（CMK）を使用
   - キーローテーションを有効化
   - 適切なキーポリシーの設定

4. **暗号化フロー**
   ```
   1. CMKからデータキーを生成（GenerateDataKey）
   2. JSON データ → tar.gz圧縮
   3. 平文データキーでtar.gzを暗号化
   4. 暗号化されたデータキー + 暗号化されたデータを保存
   5. 平文データキーをメモリから削除
   ```

5. **復号化フロー**
   ```
   1. 暗号化ファイルから暗号化データキーと暗号化データを読み込み
   2. 暗号化データキーをKMSで復号化（Decrypt）
   3. 復号化されたデータキーで暗号化データを復号化
   4. tar.gz展開 → JSON復元
   5. 復号化されたデータキーをメモリから削除
   ```

6. **データキー管理**
   - 各バックアップ操作で新しいデータキーを生成
   - データキーの再利用はしない
   - データキーはバックアップファイルと一緒に保存

### 実装設計

#### 1. 設定拡張
```go
type KMSConfig struct {
    Enabled   bool   `json:"enabled"`
    KeyID     string `json:"key_id"`
    Region    string `json:"region"`
}

type Config struct {
    // 既存フィールド...
    KMS KMSConfig `json:"kms"`
}
```

#### 2. 暗号化インターフェース
```go
// DataKeyInfo はデータキー情報を保持する
type DataKeyInfo struct {
    PlaintextKey []byte // 平文データキー（暗号化処理後は即座に削除）
    EncryptedKey []byte // 暗号化されたデータキー（保存用）
}

// EncryptedData は暗号化されたデータを保持する
type EncryptedData struct {
    EncryptedDataKey []byte // 暗号化されたデータキー
    EncryptedContent []byte // 暗号化されたコンテンツ
    IV               []byte // 初期化ベクター
}

type Encryptor interface {
    // データキーを生成する
    GenerateDataKey(ctx context.Context) (*DataKeyInfo, error)
    
    // データキーでデータを暗号化する
    EncryptWithDataKey(data []byte, plaintextKey []byte) (*EncryptedData, error)
    
    // 暗号化されたデータを復号化する
    Decrypt(ctx context.Context, encryptedData *EncryptedData) ([]byte, error)
}

type KMSEncryptor struct {
    client *kms.Client
    keyID  string
}
```

#### 3. Storage インターフェース拡張
```go
type Storage interface {
    SaveJSON(ctx context.Context, prefix, userPoolID string, filename string, data interface{}) error
    ReadFile(ctx context.Context, key string) ([]byte, error)
    CreateTarGz(ctx context.Context, prefix string) error
    GetBackupData() *BackupData
    // 新規追加
    SetEncryptor(encryptor Encryptor)
}
```

#### 4. 暗号化統合
- `CreateTarGz` メソッド内でtar.gz作成後にKMS暗号化を実行
- `ReadFile` メソッド内でファイル読み込み後にKMS復号化を実行
- ローカルストレージとS3ストレージ両方でサポート

#### 5. ファイル形式
暗号化されたバックアップファイルは以下の形式で保存される：

```
| 4 bytes | 4 bytes | N bytes | M bytes |
| IV長    | 暗号化DK長 | IV     | 暗号化DK | 暗号化データ |
```

- IV長: 初期化ベクターの長さ（4バイト）
- 暗号化DK長: 暗号化されたデータキーの長さ（4バイト）
- IV: AES-GCMの初期化ベクター（12バイト）
- 暗号化DK: 暗号化されたデータキー
- 暗号化データ: AES-256-GCMで暗号化されたtar.gzデータ

## データキー生成コマンド設計

### CLIコマンド追加

#### generate-datakey サブコマンド
```bash
# 基本使用法
cognito-backup generate-datakey --kms-key-id <KMS_KEY_ID> [options]

# オプション
--kms-key-id, -k     KMSキーID（必須）
--output, -o         出力ファイルパス（デフォルト: stdout）
--format, -f         出力形式（json|base64）デフォルト: json
--spec              データキー仕様（AES_256|AES_128）デフォルト: AES_256

# 使用例
cognito-backup generate-datakey --kms-key-id alias/cognito-backup-key
cognito-backup generate-datakey --kms-key-id arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012 --output datakey.json
```

#### 出力形式
JSON形式での出力例：
```json
{
  "kms_key_id": "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012",
  "encrypted_data_key": "AQIDAHhqBXXXXXXXX...",
  "plaintext_data_key": "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
  "key_spec": "AES_256",
  "generated_at": "2024-01-01T12:00:00Z"
}
```

Base64形式での出力例：
```
# 暗号化されたデータキー（Base64）
AQIDAHhqBXXXXXXXX...

# 平文データキー（Base64）- 注意：セキュアに扱うこと
XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
```

#### テストコマンド
```bash
# データキーのテスト生成（実際のKMS呼び出しなし）
cognito-backup generate-datakey --test --output test-datakey.json

# 生成されたデータキーの検証
cognito-backup validate-datakey --input datakey.json --kms-key-id <KMS_KEY_ID>
```

### 実装段階

#### Phase 1: 基本暗号化機能
- KMSEncryptorの実装
- Storageインターフェースの拡張
- 設定ファイルの拡張

#### Phase 2: 統合とテスト
- LocalStorageとS3Storageへの統合
- エラーハンドリングの強化
- 単体テストの追加

#### Phase 3: 下位互換性とマイグレーション
- 暗号化フラグによる段階的移行
- 既存の非暗号化バックアップのサポート継続

## 結果

### メリット
- **セキュリティ向上**: バックアップデータの暗号化により機密情報を保護
- **コンプライアンス**: データ保護規制への準拠
- **監査可能性**: KMSのログ機能によるアクセス追跡
- **キー管理**: AWSによる安全なキー管理とローテーション
- **パフォーマンス**: エンベロープ暗号化により大容量データの高速処理
- **コスト効率**: データキーによる暗号化でKMS API呼び出し回数を削減

### デメリット
- **複雑性の増加**: 暗号化/復号化処理の追加
- **パフォーマンス影響**: 暗号化処理による若干の性能低下
- **コスト**: KMS使用料金の発生
- **依存関係**: AWS KMSサービスへの依存

### リスク軽減策
- 適切なエラーハンドリングの実装
- KMSキーへのアクセス権限の適切な管理
- 暗号化されていないデータの段階的移行サポート
- バックアップとリストア処理のテスト強化

## 実装考慮事項

### セキュリティ
- KMSキーポリシーでアクセス制御を適切に設定
- 復号化権限を必要最小限に制限
- CloudTrailでKMS操作をログ記録

### 運用
- キーローテーションスケジュールの設定
- 障害時の復旧手順の整備
- モニタリングとアラートの設定

### 下位互換性
- 暗号化の有効/無効を設定で制御
- 既存の非暗号化バックアップの読み込みサポート
- 段階的移行のためのマイグレーションツール

## 代替案

### 1. アプリケーションレベル暗号化
- 独自の暗号化ライブラリを使用
- キー管理が複雑になる

### 2. S3サーバーサイド暗号化のみ
- S3-SSEを使用
- ローカルストレージでは暗号化されない

### 3. 暗号化なし
- 現状維持
- セキュリティリスクが残る

## 参考資料
- [AWS KMS Developer Guide](https://docs.aws.amazon.com/kms/latest/developerguide/)
- [AWS SDK for Go KMS Package](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/kms)
- [S3 Server-Side Encryption](https://docs.aws.amazon.com/AmazonS3/latest/userguide/serv-side-encryption.html)