package safebackup

import "errors"

var (
	// ErrInvalidConfig は設定が無効な場合のエラー
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrBackupFailed はバックアップに失敗した場合のエラー
	ErrBackupFailed = errors.New("backup failed")

	// ErrCleaningTimeout はクリーニングがタイムアウトした場合のエラー
	ErrCleaningTimeout = errors.New("cleaning timeout")
)
