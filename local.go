package safebackup

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	cleaner "github.com/ideamans/go-backup-cleaner"
)

// LocalBackupSession はローカルファイルシステムへのバックアップセッション実装
type LocalBackupSession struct {
	config           LocalBackupSessionConfig
	accumulatedSize  int64          // 累積ファイルサイズ（atomic）
	cleaningMutex    sync.Mutex     // クリーニング排他制御
	isCleaningActive atomic.Bool    // クリーニング実行中フラグ
	cleaningDone     chan struct{}  // クリーニング完了通知
	wg               sync.WaitGroup // 全処理の完了待機
}

// NewLocalBackupSession はローカルバックアップセッションインスタンスを作成
// 作成時に自動的に容量チェックを行い、必要に応じてクリーニングを開始する
func NewLocalBackupSession(config LocalBackupSessionConfig) (*LocalBackupSession, error) {
	// 設定の検証
	if err := validateLocalConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// デフォルト値の設定
	if config.CheckInterval == 0 {
		config.CheckInterval = 1024 * 1024 * 1024 // 1GB
	}

	session := &LocalBackupSession{
		config:       config,
		cleaningDone: make(chan struct{}),
	}

	// ルートディレクトリが存在しない場合は作成
	if err := os.MkdirAll(config.RootDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create root directory: %w", err)
	}

	// 初期容量チェックとクリーニング
	diskInfo, err := config.CleaningConfig.DiskInfo.GetDiskUsage(config.RootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk usage: %w", err)
	}

	if diskInfo.Free < config.FreeSpaceThreshold {
		// 空き容量が不足している場合、即座にクリーニングを開始
		session.isCleaningActive.Store(true)
		session.wg.Add(1)

		go func() {
			defer session.wg.Done()
			session.performCleaning()
		}()
	}

	return session, nil
}

// Save はファイルをバックアップディレクトリに保存する
func (s *LocalBackupSession) Save(localFilePath, relativePath string) error {
	// 入力検証
	if localFilePath == "" || relativePath == "" {
		return fmt.Errorf("%w: empty file path", ErrInvalidConfig)
	}

	// ソースファイルの情報を取得
	srcInfo, err := os.Stat(localFilePath)
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	if !srcInfo.Mode().IsRegular() {
		return fmt.Errorf("%w: source is not a regular file", ErrInvalidConfig)
	}

	// 宛先パスの構築
	destPath := filepath.Join(s.config.RootDir, relativePath)
	destDir := filepath.Dir(destPath)

	// 宛先ディレクトリの作成
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// ファイルのコピー
	if err := s.copyFile(localFilePath, destPath); err != nil {
		return fmt.Errorf("%w: %v", ErrBackupFailed, err)
	}

	// ファイルサイズを累積
	fileSize := srcInfo.Size()
	newAccumulatedSize := atomic.AddInt64(&s.accumulatedSize, fileSize)

	// 累積サイズがチェック間隔を超えたら容量チェック
	if newAccumulatedSize >= int64(s.config.CheckInterval) {
		s.checkAndCleanIfNeeded()
	}

	return nil
}

// WaitForCompletion はすべての処理の完了を待つ
func (s *LocalBackupSession) WaitForCompletion(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("%w: %v", ErrCleaningTimeout, ctx.Err())
	case <-done:
		return nil
	}
}

// Close はリソースをクリーンアップする
func (s *LocalBackupSession) Close() error {
	// クリーニング完了通知チャネルをクローズ
	close(s.cleaningDone)
	return nil
}

// copyFile はファイルをコピーする
func (s *LocalBackupSession) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer func() {
		_ = sourceFile.Close()
	}()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() {
		_ = destFile.Close()
	}()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		_ = os.Remove(dst)
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// ファイルの権限をコピー
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	if err := destFile.Chmod(srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return destFile.Sync()
}

// checkAndCleanIfNeeded は容量チェックを行い、必要に応じてクリーニングを開始する
func (s *LocalBackupSession) checkAndCleanIfNeeded() {
	// 既にクリーニング中なら何もしない
	if s.isCleaningActive.Load() {
		return
	}

	// 排他制御
	s.cleaningMutex.Lock()
	defer s.cleaningMutex.Unlock()

	// 再度チェック（ダブルチェック）
	if s.isCleaningActive.Load() {
		return
	}

	// ディスク容量チェック
	diskInfo, err := s.config.CleaningConfig.DiskInfo.GetDiskUsage(s.config.RootDir)
	if err != nil {
		// エラーの場合はログに記録するが処理は継続
		return
	}

	// 空き容量が閾値を下回っている場合
	if diskInfo.Free < s.config.FreeSpaceThreshold {
		s.isCleaningActive.Store(true)
		s.wg.Add(1)

		go func() {
			defer s.wg.Done()
			s.performCleaning()
		}()
	}
}

// performCleaning は実際のクリーニング処理を実行する
func (s *LocalBackupSession) performCleaning() {
	defer func() {
		s.isCleaningActive.Store(false)
		// 累積サイズをリセット
		atomic.StoreInt64(&s.accumulatedSize, 0)
	}()

	// クリーニング設定の準備
	config := s.config.CleaningConfig

	// 目標使用率の計算（目標空き容量から逆算）
	diskInfo, err := config.DiskInfo.GetDiskUsage(s.config.RootDir)
	if err != nil {
		return
	}

	var targetUsedSpace uint64
	if s.config.TargetFreeSpace < diskInfo.Total {
		targetUsedSpace = diskInfo.Total - s.config.TargetFreeSpace
	} else {
		targetUsedSpace = 0
	}
	targetUsagePercent := float64(targetUsedSpace) / float64(diskInfo.Total) * 100

	// MaxUsagePercentを設定
	if config.MaxUsagePercent == nil {
		config.MaxUsagePercent = &targetUsagePercent
	}

	// クリーニング実行
	report, err := cleaner.CleanBackup(s.config.RootDir, config)
	if err != nil {
		// エラーはログに記録するが処理は継続
		return
	}
	_ = report // レポートは今回は使用しない
}

// validateLocalConfig はローカルバックアップ設定を検証する
func validateLocalConfig(config LocalBackupSessionConfig) error {
	if config.RootDir == "" {
		return fmt.Errorf("%w: root directory is required", ErrInvalidConfig)
	}

	if config.FreeSpaceThreshold <= 0 {
		return fmt.Errorf("%w: free space threshold must be positive", ErrInvalidConfig)
	}

	if config.TargetFreeSpace <= 0 {
		return fmt.Errorf("%w: target free space must be positive", ErrInvalidConfig)
	}

	if config.TargetFreeSpace <= config.FreeSpaceThreshold {
		return fmt.Errorf("%w: target free space must be greater than threshold", ErrInvalidConfig)
	}

	return nil
}
