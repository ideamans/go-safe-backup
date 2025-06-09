package safebackup

import (
	cleaner "github.com/ideamans/go-backup-cleaner"
)

// LocalBackupSessionConfig はローカルバックアップセッションの設定
type LocalBackupSessionConfig struct {
	// RootDir はバックアップのルートディレクトリ
	RootDir string

	// FreeSpaceThreshold はクリーニングを開始する空き容量（バイト）
	FreeSpaceThreshold uint64

	// TargetFreeSpace はクリーニング後の目標空き容量（バイト）
	TargetFreeSpace uint64

	// CheckInterval は空き容量チェックを行うファイルサイズの累積間隔（デフォルト: 1GB）
	CheckInterval uint64

	// CleaningConfig はgo-backup-cleanerの設定
	CleaningConfig cleaner.CleaningConfig
}

// S3BackupSessionConfig はS3バックアップセッションの設定
type S3BackupSessionConfig struct {
	// AWS認証情報
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string // オプション

	// S3設定
	Bucket   string
	Prefix   string // バックアップのプレフィックス（オプション）
	Endpoint string // カスタムエンドポイント（オプション）

	// ACL設定
	ACL string // デフォルト: private
}
