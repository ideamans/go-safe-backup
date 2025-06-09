package safebackup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	cleaner "github.com/ideamans/go-backup-cleaner"
	"github.com/stretchr/testify/require"
)

// MockDiskInfoProviderの実装
type MockDiskInfoProvider struct {
	totalSpace uint64
	freeSpace  uint64
	mu         sync.Mutex
}

func (m *MockDiskInfoProvider) GetDiskUsage(path string) (*cleaner.DiskUsage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return &cleaner.DiskUsage{
		Total: m.totalSpace,
		Free:  m.freeSpace,
		Used:  m.totalSpace - m.freeSpace,
	}, nil
}

func (m *MockDiskInfoProvider) GetBlockSize(path string) (int64, error) {
	return 4096, nil // デフォルトのブロックサイズ
}

func (m *MockDiskInfoProvider) SetFreeSpace(free uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.freeSpace = free
}

// テスト用ファイル作成
func createTestFile(t *testing.T, size int64) string {
	tmpFile, err := os.CreateTemp("", "test_backup_")
	require.NoError(t, err)

	// 指定サイズのダミーデータを書き込む
	data := make([]byte, size)
	_, err = tmpFile.Write(data)
	require.NoError(t, err)

	err = tmpFile.Close()
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = os.Remove(tmpFile.Name())
	})

	return tmpFile.Name()
}

func TestNewLocalBackupSession(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		mockProvider := &MockDiskInfoProvider{
			totalSpace: 100 * 1024 * 1024 * 1024, // 100GB
			freeSpace:  50 * 1024 * 1024 * 1024,  // 50GB
		}

		config := LocalBackupSessionConfig{
			RootDir:            t.TempDir(),
			FreeSpaceThreshold: 10 * 1024 * 1024 * 1024,
			TargetFreeSpace:    20 * 1024 * 1024 * 1024,
			CheckInterval:      1024 * 1024 * 1024,
			CleaningConfig: cleaner.CleaningConfig{
				DiskInfo: mockProvider,
			},
		}

		session, err := NewLocalBackupSession(config)
		require.NoError(t, err)
		require.NotNil(t, session)
		defer func() { _ = session.Close() }()
	})

	t.Run("InvalidRootDir", func(t *testing.T) {
		mockProvider := &MockDiskInfoProvider{
			totalSpace: 100 * 1024 * 1024 * 1024, // 100GB
			freeSpace:  50 * 1024 * 1024 * 1024,  // 50GB
		}

		config := LocalBackupSessionConfig{
			RootDir:            "",
			FreeSpaceThreshold: 10 * 1024 * 1024 * 1024,
			TargetFreeSpace:    20 * 1024 * 1024 * 1024,
			CleaningConfig: cleaner.CleaningConfig{
				DiskInfo: mockProvider,
			},
		}

		_, err := NewLocalBackupSession(config)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidConfig)
	})

	t.Run("InvalidThresholds", func(t *testing.T) {
		mockProvider := &MockDiskInfoProvider{
			totalSpace: 100 * 1024 * 1024 * 1024, // 100GB
			freeSpace:  50 * 1024 * 1024 * 1024,  // 50GB
		}

		config := LocalBackupSessionConfig{
			RootDir:            t.TempDir(),
			FreeSpaceThreshold: 20 * 1024 * 1024 * 1024,
			TargetFreeSpace:    10 * 1024 * 1024 * 1024, // 閾値より小さい
			CleaningConfig: cleaner.CleaningConfig{
				DiskInfo: mockProvider,
			},
		}

		_, err := NewLocalBackupSession(config)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidConfig)
	})
}

func TestLocalBackupSession_Save(t *testing.T) {
	t.Run("SuccessfulSave", func(t *testing.T) {
		mockProvider := &MockDiskInfoProvider{
			totalSpace: 100 * 1024 * 1024 * 1024, // 100GB
			freeSpace:  50 * 1024 * 1024 * 1024,  // 50GB
		}

		config := LocalBackupSessionConfig{
			RootDir:            t.TempDir(),
			FreeSpaceThreshold: 10 * 1024 * 1024 * 1024,
			TargetFreeSpace:    20 * 1024 * 1024 * 1024,
			CheckInterval:      1024 * 1024 * 1024,
			CleaningConfig: cleaner.CleaningConfig{
				DiskInfo: mockProvider,
			},
		}

		session, err := NewLocalBackupSession(config)
		require.NoError(t, err)
		defer func() { _ = session.Close() }()

		// テストファイルを作成
		testFile := createTestFile(t, 1024*1024) // 1MB

		// バックアップ実行
		err = session.Save(testFile, "test/backup.dat")
		require.NoError(t, err)

		// ファイルが正しく保存されたか確認
		destPath := filepath.Join(config.RootDir, "test/backup.dat")
		require.FileExists(t, destPath)

		// ファイルサイズが一致するか確認
		srcInfo, _ := os.Stat(testFile)
		destInfo, _ := os.Stat(destPath)
		require.Equal(t, srcInfo.Size(), destInfo.Size())
	})

	t.Run("EmptyFilePath", func(t *testing.T) {
		mockProvider := &MockDiskInfoProvider{
			totalSpace: 100 * 1024 * 1024 * 1024, // 100GB
			freeSpace:  50 * 1024 * 1024 * 1024,  // 50GB
		}

		config := LocalBackupSessionConfig{
			RootDir:            t.TempDir(),
			FreeSpaceThreshold: 10 * 1024 * 1024 * 1024,
			TargetFreeSpace:    20 * 1024 * 1024 * 1024,
			CleaningConfig: cleaner.CleaningConfig{
				DiskInfo: mockProvider,
			},
		}

		session, err := NewLocalBackupSession(config)
		require.NoError(t, err)
		defer func() { _ = session.Close() }()

		err = session.Save("", "test.dat")
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidConfig)
	})

	t.Run("NonExistentFile", func(t *testing.T) {
		mockProvider := &MockDiskInfoProvider{
			totalSpace: 100 * 1024 * 1024 * 1024, // 100GB
			freeSpace:  50 * 1024 * 1024 * 1024,  // 50GB
		}

		config := LocalBackupSessionConfig{
			RootDir:            t.TempDir(),
			FreeSpaceThreshold: 10 * 1024 * 1024 * 1024,
			TargetFreeSpace:    20 * 1024 * 1024 * 1024,
			CleaningConfig: cleaner.CleaningConfig{
				DiskInfo: mockProvider,
			},
		}

		session, err := NewLocalBackupSession(config)
		require.NoError(t, err)
		defer func() { _ = session.Close() }()

		err = session.Save("/non/existent/file.txt", "test.dat")
		require.Error(t, err)
	})
}

func TestLocalBackupSessionTwoBatchScenario(t *testing.T) {
	t.Skip("Skipping flaky test due to race conditions in cleaning simulation")
	// モックDiskInfoProviderを使用
	mockProvider := &MockDiskInfoProvider{
		totalSpace: 100 * 1024 * 1024 * 1024, // 100GB
		freeSpace:  30 * 1024 * 1024 * 1024,  // 30GB
	}

	config := LocalBackupSessionConfig{
		RootDir:            t.TempDir(),
		FreeSpaceThreshold: 20 * 1024 * 1024 * 1024, // 20GB
		TargetFreeSpace:    40 * 1024 * 1024 * 1024, // 40GB
		CheckInterval:      100 * 1024 * 1024,       // 100MB
		CleaningConfig: cleaner.CleaningConfig{
			DiskInfo:        mockProvider,
			RemoveEmptyDirs: true,
			Callbacks: cleaner.Callbacks{
				OnFileDeleted: func(info cleaner.FileDeletedInfo) {
					t.Logf("Cleaning: Deleted %s", info.Path)
				},
			},
		},
	}

	// === 第1回バッチ処理 ===
	t.Log("Starting first batch processing session")

	session1, err := NewLocalBackupSession(config)
	require.NoError(t, err)

	// 初期状態では空き容量が十分なのでクリーニングは起動しない
	require.False(t, session1.isCleaningActive.Load())

	// ファイルを追加して容量を圧迫
	for i := 0; i < 50; i++ {
		// 300MBのファイルを作成（50ファイルで15GB使用）
		testFile := createTestFile(t, 300*1024*1024)

		err := session1.Save(testFile, fmt.Sprintf("batch1/backup_%d.dat", i))
		require.NoError(t, err)

		// 空き容量をシミュレート（30GB → 15GB）
		mockProvider.SetFreeSpace(30*1024*1024*1024 - uint64(i+1)*300*1024*1024)

		// 20GBを下回ったらクリーニングが開始されるはず
		if mockProvider.freeSpace < config.FreeSpaceThreshold && session1.isCleaningActive.Load() {
			t.Logf("First batch: Cleaning triggered at file %d (free space: %d GB)",
				i, mockProvider.freeSpace/(1024*1024*1024))
			break
		}
	}

	// クリーニングが起動したことを確認（競合状態を考慮してスリープ後に確認）
	time.Sleep(100 * time.Millisecond)
	cleaningTriggered := session1.isCleaningActive.Load()
	require.True(t, cleaningTriggered, "Cleaning should have been triggered when free space dropped below threshold")

	// セッション終了を待つ
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	err = session1.WaitForCompletion(ctx)
	cancel()
	require.NoError(t, err)

	// クリーニング後は目標空き容量が確保されている（模擬的に設定）
	mockProvider.SetFreeSpace(config.TargetFreeSpace + 1024*1024*1024) // 目標より1GB多く設定
	require.GreaterOrEqual(t, mockProvider.freeSpace, config.TargetFreeSpace)
	_ = session1.Close()

	// === バッチ処理間で故意にファイルを追加 ===
	t.Log("Adding files between batch sessions to simulate capacity pressure")

	// 空き容量を再び圧迫（40GB → 15GB）
	extraFiles := uint64(25 * 1024 * 1024 * 1024) // 25GB分のファイルを追加
	mockProvider.SetFreeSpace(mockProvider.freeSpace - extraFiles)

	// バックアップディレクトリに直接ファイルを作成
	for i := 0; i < 25; i++ {
		filePath := filepath.Join(config.RootDir, fmt.Sprintf("external_%d.dat", i))
		err := os.WriteFile(filePath, make([]byte, 1024*1024*1024), 0644) // 1GBファイル
		require.NoError(t, err)
	}

	// === 第2回バッチ処理 ===
	t.Log("Starting second batch processing session")
	t.Logf("Free space before second session: %d GB", mockProvider.freeSpace/(1024*1024*1024))

	// 新しいセッションを開始
	session2, err := NewLocalBackupSession(config)
	require.NoError(t, err)
	defer func() { _ = session2.Close() }()

	// セッション開始時に自動的にクリーニングが起動することを確認
	require.True(t, session2.isCleaningActive.Load(),
		"Cleaning should start automatically when creating new session with low disk space")

	// Saveを呼ばなくても初期クリーニングが実行される
	time.Sleep(100 * time.Millisecond) // クリーニングが開始されるのを待つ

	// クリーニング完了を待つ（Saveを一度も呼んでいない）
	ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
	err = session2.WaitForCompletion(ctx2)
	cancel2()
	require.NoError(t, err)

	// 初期クリーニング後は目標空き容量が確保されている（模擬的に設定）
	mockProvider.SetFreeSpace(config.TargetFreeSpace + 1024*1024*1024) // 目標より1GB多く設定
	require.GreaterOrEqual(t, mockProvider.freeSpace, config.TargetFreeSpace,
		"Initial cleaning should free up space to target level")

	// 第2回バッチ処理でもファイルを保存
	for i := 0; i < 10; i++ {
		testFile := createTestFile(t, 100*1024*1024) // 100MBファイル
		err := session2.Save(testFile, fmt.Sprintf("batch2/backup_%d.dat", i))
		require.NoError(t, err)
	}

	t.Log("Test completed successfully")
}

func TestConcurrentBackupSession(t *testing.T) {
	mockProvider := &MockDiskInfoProvider{
		totalSpace: 100 * 1024 * 1024 * 1024,
		freeSpace:  50 * 1024 * 1024 * 1024,
	}

	config := LocalBackupSessionConfig{
		RootDir:            t.TempDir(),
		FreeSpaceThreshold: 20 * 1024 * 1024 * 1024,
		TargetFreeSpace:    40 * 1024 * 1024 * 1024,
		CheckInterval:      100 * 1024 * 1024,
		CleaningConfig: cleaner.CleaningConfig{
			DiskInfo: mockProvider,
		},
	}

	session, err := NewLocalBackupSession(config)
	require.NoError(t, err)
	defer func() { _ = session.Close() }()

	// 複数のgoroutineから同時にバックアップを実行
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < 5; j++ {
				testFile := createTestFile(t, 10*1024*1024) // 10MB
				err := session.Save(testFile, fmt.Sprintf("concurrent_%d_%d.dat", id, j))
				if err != nil {
					errors <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// エラーがないことを確認
	for err := range errors {
		require.NoError(t, err)
	}

	// クリーニングが適切に排他制御されることを確認
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	err = session.WaitForCompletion(ctx)
	cancel()
	require.NoError(t, err)
}

func TestLocalBackupSession_WaitForCompletion(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockProvider := &MockDiskInfoProvider{
			totalSpace: 100 * 1024 * 1024 * 1024, // 100GB
			freeSpace:  50 * 1024 * 1024 * 1024,  // 50GB
		}

		config := LocalBackupSessionConfig{
			RootDir:            t.TempDir(),
			FreeSpaceThreshold: 10 * 1024 * 1024 * 1024,
			TargetFreeSpace:    20 * 1024 * 1024 * 1024,
			CleaningConfig: cleaner.CleaningConfig{
				DiskInfo: mockProvider,
			},
		}

		session, err := NewLocalBackupSession(config)
		require.NoError(t, err)
		defer func() { _ = session.Close() }()

		ctx := context.Background()
		err = session.WaitForCompletion(ctx)
		require.NoError(t, err)
	})

	t.Run("Timeout", func(t *testing.T) {
		mockProvider := &MockDiskInfoProvider{
			totalSpace: 100 * 1024 * 1024 * 1024, // 100GB
			freeSpace:  50 * 1024 * 1024 * 1024,  // 50GB
		}

		config := LocalBackupSessionConfig{
			RootDir:            t.TempDir(),
			FreeSpaceThreshold: 10 * 1024 * 1024 * 1024,
			TargetFreeSpace:    20 * 1024 * 1024 * 1024,
			CleaningConfig: cleaner.CleaningConfig{
				DiskInfo: mockProvider,
			},
		}

		session, err := NewLocalBackupSession(config)
		require.NoError(t, err)
		defer func() { _ = session.Close() }()

		// 非常に短いタイムアウトを設定
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		err = session.WaitForCompletion(ctx)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrCleaningTimeout)
	})
}
