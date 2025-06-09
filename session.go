package safebackup

import (
	"context"
)

// BackupSession はバックアップセッションの共通インターフェース
type BackupSession interface {
	// Save はファイルをバックアップ先に保存する
	// localFilePath: バックアップ元のファイルパス
	// relativePath: バックアップ先での相対パス
	Save(localFilePath, relativePath string) error

	// WaitForCompletion はバックアップ処理とクリーニング処理の完了を待つ
	// ctx: タイムアウト制御用のコンテキスト
	WaitForCompletion(ctx context.Context) error

	// Close はリソースをクリーンアップする
	Close() error
}
