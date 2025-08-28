# go-safe-backup

[![CI](https://github.com/ideamans/go-safe-backup/workflows/CI/badge.svg)](https://github.com/ideamans/go-safe-backup/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/ideamans/go-safe-backup.svg)](https://pkg.go.dev/github.com/ideamans/go-safe-backup)
[![License](https://img.shields.io/github/license/ideamans/go-safe-backup.svg)](LICENSE)

Go言語用の容量安全バックアップライブラリ。ローカルバックアップの自動ディスク容量管理を提供し、ローカルファイルシステムとAmazon S3の両方をバックアップ先としてサポートします。

## 機能

- **自動ディスク容量管理**: ディスク使用状況を監視し、容量が少なくなると古いバックアップを自動的にクリーンアップ
- **セッションベースのアーキテクチャ**: ライフサイクル管理を備えたクリーンなAPI
- **複数バックエンド**: ローカルファイルシステムとS3互換ストレージをサポート
- **並行処理**: 設定可能な並行性を持つスレッドセーフな操作
- **包括的なテスト**: ユニットテスト、統合テスト、モックプロバイダー

## インストール

```bash
go get github.com/ideamans/go-safe-backup
```

## クイックスタート

### ローカルバックアップ

```go
package main

import (
    "context"
    "time"
    
    "github.com/ideamans/go-safe-backup"
    cleaner "github.com/ideamans/go-backup-cleaner"
)

func main() {
    config := safebackup.LocalBackupSessionConfig{
        RootDir:            "/path/to/backup/directory",
        FreeSpaceThreshold: 10 * 1024 * 1024 * 1024, // 10GB
        TargetFreeSpace:    20 * 1024 * 1024 * 1024, // 20GB
        CheckInterval:      1024 * 1024 * 1024,       // 1GB
        CleaningConfig: cleaner.CleaningConfig{
            RemoveEmptyDirs: true,
        },
    }
    
    session, err := safebackup.NewLocalBackupSession(config)
    if err != nil {
        panic(err)
    }
    defer session.Close()
    
    // ファイルをバックアップ
    err = session.Save("/path/to/source/file.txt", "backups/file.txt")
    if err != nil {
        panic(err)
    }
    
    // 完了を待つ
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
    defer cancel()
    
    err = session.WaitForCompletion(ctx)
    if err != nil {
        panic(err)
    }
}
```

### S3バックアップ

```go
package main

import (
    "context"
    "time"
    
    "github.com/ideamans/go-safe-backup"
)

func main() {
    config := safebackup.S3BackupSessionConfig{
        Region:          "us-east-1",
        AccessKeyID:     "your-access-key",
        SecretAccessKey: "your-secret-key",
        Bucket:          "your-backup-bucket",
        Prefix:          "backups/",
        ACL:             "private",
    }
    
    session, err := safebackup.NewS3BackupSession(config)
    if err != nil {
        panic(err)
    }
    defer session.Close()
    
    // ファイルをバックアップ
    err = session.Save("/path/to/source/file.txt", "2024/01/file.txt")
    if err != nil {
        panic(err)
    }
    
    // 完了を待つ
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
    defer cancel()
    
    err = session.WaitForCompletion(ctx)
    if err != nil {
        panic(err)
    }
}
```

## APIリファレンス

### BackupSessionインターフェース

```go
type BackupSession interface {
    // Saveはファイルを宛先にバックアップします
    Save(localFilePath, relativePath string) error
    
    // WaitForCompletionはすべてのバックアップとクリーニング操作の完了を待ちます
    WaitForCompletion(ctx context.Context) error
    
    // Closeはリソースをクリーンアップします
    Close() error
}
```

### ローカルバックアップ設定

```go
type LocalBackupSessionConfig struct {
    // RootDirはバックアップのルートディレクトリ
    RootDir string
    
    // FreeSpaceThresholdはクリーンアップをトリガーする前の最小空き容量（バイト）
    FreeSpaceThreshold uint64
    
    // TargetFreeSpaceはクリーンアップ後の目標空き容量（バイト）
    TargetFreeSpace uint64
    
    // CheckIntervalは容量チェックのためのファイルサイズ累積間隔（デフォルト: 1GB）
    CheckInterval uint64
    
    // CleaningConfigはgo-backup-cleanerの設定
    CleaningConfig cleaner.CleaningConfig
}
```

### S3バックアップ設定

```go
type S3BackupSessionConfig struct {
    // AWS認証情報
    Region          string
    AccessKeyID     string
    SecretAccessKey string
    SessionToken    string // オプション
    
    // S3設定
    Bucket   string
    Prefix   string // オプションのバックアップ接頭辞
    Endpoint string // オプションのカスタムエンドポイント（MinIO等用）
    
    // ACL設定
    ACL string // デフォルト: private
}
```

## 開発

### 前提条件

- Go 1.22以降
- Docker（統合テスト用）

### 開発コマンド

```bash
# すべてのテストを実行
go test -v ./...

# ユニットテストのみ実行（統合テストをスキップ）
go test -v -short ./...

# レース検出付きでテストを実行
go test -v -race ./...

# 統合テストを実行（Dockerが必要）
go test -v -tags=integration ./...

# テストカバレッジを生成
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# コードをフォーマット
go fmt ./...

# 静的解析を実行
go vet ./...

# 依存関係をクリーンアップ
go mod tidy
```

### MinIOでのテスト

統合テストはDockerを使用してMinIOコンテナを自動的に起動し、S3機能をテストします：

```bash
# 統合テストを実行
go test -v -tags=integration -run Integration ./...
```

### プロジェクト構造

```
.
├── session.go          # 共通インターフェース定義
├── local.go           # ローカルファイルシステム実装
├── s3.go              # S3/MinIO実装
├── types.go           # 共有型定義
├── errors.go          # エラー型定義
├── local_test.go      # ローカルバックアップテスト
├── s3_test.go         # S3バックアップテスト
├── integration_test.go # Dockerを使用した統合テスト
└── examples/          # 使用例
    ├── local/
    └── s3/
```

## アーキテクチャ

### セッションベースの設計

このライブラリは、各バックアップ操作がライフサイクルを管理するセッションを作成するセッションベースのアーキテクチャを使用します：

1. **セッション作成**: 設定を検証し、リソースをセットアップ
2. **ファイル操作**: 自動容量監視を備えたスレッドセーフなバックアップ操作
3. **クリーンアップ管理**: 容量が少なくなった時の古いバックアップの自動クリーンアップトリガー
4. **リソースクリーンアップ**: セッション終了時の適切なリソースクリーンアップ

### ローカルバックアップ機能

- **自動容量監視**: セッション作成時と操作中にディスク容量をチェック
- **累積サイズ追跡**: スレッドセーフなサイズ追跡のためのアトミック操作を使用
- **設定可能なしきい値**: クリーンアップトリガーと目標空き容量の個別しきい値
- **go-backup-cleanerとの統合**: インテリジェントなクリーンアップのためにgo-backup-cleanerライブラリを活用

### S3バックアップ機能

- **並行アップロード**: 設定可能なACLを持つ非同期ファイルアップロード
- **カスタムエンドポイント**: MinIOおよび他のS3互換サービスのサポート
- **AWS SDK統合**: AWS S3およびIAMロールとの完全な互換性

## エラー処理

ライブラリはさまざまなシナリオに対して特定のエラー型を定義しています：

```go
var (
    ErrInvalidConfig   = errors.New("invalid configuration")
    ErrBackupFailed    = errors.New("backup failed")
    ErrCleaningTimeout = errors.New("cleaning timeout")
)
```

## コントリビューション

1. リポジトリをフォーク
2. フィーチャーブランチを作成
3. 新機能のテストを追加
4. すべてのテストが合格することを確認
5. プルリクエストを送信

## ライセンス

[ここにライセンス情報を追加]

## 依存関係

- [github.com/ideamans/go-backup-cleaner](https://github.com/ideamans/go-backup-cleaner) - 自動ディスク容量管理
- [github.com/aws/aws-sdk-go](https://github.com/aws/aws-sdk-go) - AWS S3操作
- [github.com/stretchr/testify](https://github.com/stretchr/testify) - テストフレームワーク
- [github.com/ory/dockertest/v3](https://github.com/ory/dockertest/v3) - コンテナを使用した統合テスト