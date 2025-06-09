## 10. 今後の拡張性

1. **他のストレージバックエンドの追加**
   - Google Cloud Storage
   - Azure Blob Storage
   - SFTP/FTP

2. **圧縮機能の追加**
   - バックアップ時の自動圧縮
   - 圧縮レベルの設定

3. **暗号化機能の追加**
   - クライアントサイド暗号化
   - 鍵管理機能

4. **メトリクスとモニタリング**
   - バックアップ統計情報
   - Prometheusメトリクス対応

5. **レプリケーション機能**
   - 複数のバックアップ先への同時保存
   - フェイルオーバー機能# 容量安全型バックアップライブラリ設計書

## 1. 概要

本ライブラリ「safebackup」は、ローカルファイルシステムとAmazon S3の両方に対応し、ディスク容量を考慮した安全なバックアップ処理を提供します。ローカルファイルシステムでは、`go-backup-cleaner`を利用して自動的に古いバックアップを削除し、ディスク容量を確保します。

重要な特徴として、ローカルバックアップではセッション開始時（インスタンス作成時）に自動的に容量チェックを行い、必要に応じてクリーニングを開始します。これにより、前回のセッションで容量が逼迫した状態でも、新しいセッションを安全に開始できます。

## 2. パッケージ構成

```
safebackup/
├── session.go         # 共通インターフェース定義
├── local.go           # ローカルファイルシステム実装
├── s3.go              # S3実装
├── types.go           # 共通型定義
├── errors.go          # エラー定義
└── session_test.go    # テスト
```

## 3. インターフェース設計

### 3.1 共通インターフェース

```go
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
```

### 3.2 ローカルファイルシステム用構成

```go
package safebackup

import (
    cleaner "github.com/ideamans/go-backup-cleaner"
    "sync"
    "sync/atomic"
)

// LocalBackupSessionConfig はローカルバックアップセッションの設定
type LocalBackupSessionConfig struct {
    // RootDir はバックアップのルートディレクトリ
    RootDir string
    
    // FreeSpaceThreshold はクリーニングを開始する空き容量（バイト）
    FreeSpaceThreshold int64
    
    // TargetFreeSpace はクリーニング後の目標空き容量（バイト）
    TargetFreeSpace int64
    
    // CheckInterval は空き容量チェックを行うファイルサイズの累積間隔（デフォルト: 1GB）
    CheckInterval int64
    
    // CleaningConfig はgo-backup-cleanerの設定
    CleaningConfig cleaner.CleaningConfig
}

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
func NewLocalBackupSession(config LocalBackupSessionConfig) (*LocalBackupSession, error)
```

### 3.3 S3用構成

```go
package safebackup

import (
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/service/s3"
)

// S3BackupSessionConfig はS3バックアップセッションの設定
type S3BackupSessionConfig struct {
    // AWS認証情報
    Region          string
    AccessKeyID     string
    SecretAccessKey string
    SessionToken    string  // オプション
    
    // S3設定
    Bucket   string
    Prefix   string // バックアップのプレフィックス（オプション）
    Endpoint string // カスタムエンドポイント（オプション）
    
    // ACL設定
    ACL string // デフォルト: private
}

// S3BackupSession はS3へのバックアップセッション実装
type S3BackupSession struct {
    config   S3BackupSessionConfig
    s3Client *s3.S3
    wg       sync.WaitGroup
}

// NewS3BackupSession はS3バックアップセッションインスタンスを作成
func NewS3BackupSession(config S3BackupSessionConfig) (*S3BackupSession, error)
```

## 4. 実装詳細

### 4.1 ローカルバックアップセッションの動作フロー

1. **セッション開始時の初期クリーニング**
   - NewLocalBackupSession呼び出し時に即座に空き容量をチェック
   - 空き容量が閾値を下回っていたら自動的にクリーニングを開始
   - 前回のセッションで残された大量のファイルを事前に削除

2. **ファイル保存時の処理**
   - ファイルをバックアップディレクトリにコピー
   - ファイルサイズを累積サイズに加算（atomic操作）
   - 累積サイズがチェック間隔を超えたら空き容量をチェック
   - 空き容量が閾値を下回っていたらクリーニングを開始

3. **クリーニング処理**
   - 排他制御により同時に1つのクリーニングのみ実行
   - go-backup-cleanerを使用して目標空き容量まで古いファイルを削除
   - クリーニング完了後、累積サイズをリセット

4. **完了待機**
   - WaitForCompletionでクリーニング処理の完了を待機
   - contextによるタイムアウト制御をサポート

### 4.2 初期クリーニングの実装例

```go
func NewLocalBackupSession(config LocalBackupSessionConfig) (*LocalBackupSession, error) {
    // 設定の検証
    if err := validateConfig(config); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }
    
    session := &LocalBackupSession{
        config:       config,
        cleaningDone: make(chan struct{}),
    }
    
    // 初期容量チェックとクリーニング
    diskInfo, err := cleaner.GetDiskUsageWithProvider(config.RootDir, config.CleaningConfig.DiskInfoProvider)
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
```

### 4.3 S3バックアップセッションの動作フロー

1. **ファイル保存時の処理**
   - ファイルをS3にアップロード
   - 指定されたACLを適用
   - エラー時はリトライ（AWS SDKの機能を使用）

2. **完了待機**
   - 全アップロードの完了を待機

## 5. エラーハンドリング

```go
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
```

## 6. 使用例

### 6.1 ローカルバックアップセッション

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/yourusername/safebackup"
    cleaner "github.com/ideamans/go-backup-cleaner"
)

func main() {
    // 80%を最大使用率として設定
    maxUsage := 80.0
    
    config := safebackup.LocalBackupSessionConfig{
        RootDir:            "/backup",
        FreeSpaceThreshold: 10 * 1024 * 1024 * 1024, // 10GB
        TargetFreeSpace:    20 * 1024 * 1024 * 1024, // 20GB
        CheckInterval:      1024 * 1024 * 1024,       // 1GB
        CleaningConfig: cleaner.CleaningConfig{
            MaxUsagePercent: &maxUsage,
            RemoveEmptyDirs: true,
            Callbacks: cleaner.Callbacks{
                OnFileDeleted: func(info cleaner.FileDeletedInfo) {
                    log.Printf("Deleted: %s", info.Path)
                },
            },
        },
    }
    
    // セッション作成時に自動的に容量チェックとクリーニングが実行される
    session, err := safebackup.NewLocalBackupSession(config)
    if err != nil {
        log.Fatal(err)
    }
    defer session.Close()
    
    // ファイルをバックアップ
    err = session.Save("/data/file.txt", "2024/01/file.txt")
    if err != nil {
        log.Printf("Backup failed: %v", err)
    }
    
    // 完了を待つ
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()
    
    if err := session.WaitForCompletion(ctx); err != nil {
        log.Printf("Wait failed: %v", err)
    }
}
```

### 6.2 S3バックアップセッション

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/yourusername/safebackup"
)

func main() {
    config := safebackup.S3BackupSessionConfig{
        Region:          "ap-northeast-1",
        AccessKeyID:     "your-access-key",
        SecretAccessKey: "your-secret-key",
        Bucket:          "my-backup-bucket",
        Prefix:          "backup/",
        ACL:             "private",
    }
    
    session, err := safebackup.NewS3BackupSession(config)
    if err != nil {
        log.Fatal(err)
    }
    defer session.Close()
    
    // ファイルをバックアップ
    err = session.Save("/data/file.txt", "2024/01/file.txt")
    if err != nil {
        log.Printf("Backup failed: %v", err)
    }
    
    // 完了を待つ
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()
    
    if err := session.WaitForCompletion(ctx); err != nil {
        log.Printf("Wait failed: %v", err)
    }
}
```

## 7. テストシナリオ

### 7.1 ローカルバックアップセッションの2回バッチ処理シナリオテスト

```go
func TestLocalBackupSessionTwoBatchScenario(t *testing.T) {
    // モックDiskInfoProviderを使用
    mockProvider := &MockDiskInfoProvider{
        totalSpace: 100 * 1024 * 1024 * 1024, // 100GB
        freeSpace:  30 * 1024 * 1024 * 1024,  // 30GB
    }
    
    config := LocalBackupSessionConfig{
        RootDir:            t.TempDir(),
        FreeSpaceThreshold: 20 * 1024 * 1024 * 1024, // 20GB
        TargetFreeSpace:    40 * 1024 * 1024 * 1024, // 40GB
        CheckInterval:      100 * 1024 * 1024,        // 100MB
        CleaningConfig: cleaner.CleaningConfig{
            DiskInfoProvider: mockProvider,
            RemoveEmptyDirs:  true,
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
        mockProvider.SetFreeSpace(30*1024*1024*1024 - int64(i+1)*300*1024*1024)
        
        // 20GBを下回ったらクリーニングが開始されるはず
        if mockProvider.freeSpace < config.FreeSpaceThreshold && session1.isCleaningActive.Load() {
            t.Logf("First batch: Cleaning triggered at file %d (free space: %d GB)", 
                i, mockProvider.freeSpace/(1024*1024*1024))
            break
        }
    }
    
    // クリーニングが起動したことを確認
    require.True(t, session1.isCleaningActive.Load())
    
    // セッション終了を待つ
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    err = session1.WaitForCompletion(ctx)
    cancel()
    require.NoError(t, err)
    
    // クリーニング後は目標空き容量が確保されている
    require.GreaterOrEqual(t, mockProvider.freeSpace, config.TargetFreeSpace)
    session1.Close()
    
    // === バッチ処理間で故意にファイルを追加 ===
    t.Log("Adding files between batch sessions to simulate capacity pressure")
    
    // 空き容量を再び圧迫（40GB → 15GB）
    extraFiles := 25 * 1024 * 1024 * 1024 // 25GB分のファイルを追加
    mockProvider.SetFreeSpace(mockProvider.freeSpace - int64(extraFiles))
    
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
    defer session2.Close()
    
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
    
    // 初期クリーニング後は目標空き容量が確保されている
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

// MockDiskInfoProviderの実装
type MockDiskInfoProvider struct {
    totalSpace int64
    freeSpace  int64
    mu         sync.Mutex
}

func (m *MockDiskInfoProvider) GetDiskUsage(path string) (*cleaner.DiskInfo, error) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    return &cleaner.DiskInfo{
        Total: m.totalSpace,
        Free:  m.freeSpace,
        Used:  m.totalSpace - m.freeSpace,
    }, nil
}

func (m *MockDiskInfoProvider) SetFreeSpace(free int64) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.freeSpace = free
}
```

### 7.2 S3バックアップセッションのテスト

```go
func TestS3BackupSession(t *testing.T) {
    // S3のモッククライアントを使用
    mockS3 := &MockS3Client{
        uploadedFiles: make(map[string][]byte),
    }
    
    session := &S3BackupSession{
        config: S3BackupSessionConfig{
            Bucket: "test-bucket",
            Prefix: "backup/",
            ACL:    "private",
        },
        s3Client: mockS3,
    }
    
    // ファイルをバックアップ
    testFile := createTestFile(t, 1024*1024) // 1MB
    err := session.Save(testFile, "test.dat")
    require.NoError(t, err)
    
    // アップロードされたことを確認
    key := "backup/test.dat"
    require.Contains(t, mockS3.uploadedFiles, key)
    require.Equal(t, 1024*1024, len(mockS3.uploadedFiles[key]))
    
    // ACLが正しく設定されているか確認
    require.Equal(t, "private", mockS3.lastACL)
}

// MockS3Clientの実装
type MockS3Client struct {
    uploadedFiles map[string][]byte
    lastACL       string
    mu            sync.Mutex
}

func (m *MockS3Client) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    // ファイル内容を読み込み
    body, err := io.ReadAll(input.Body)
    if err != nil {
        return nil, err
    }
    
    m.uploadedFiles[*input.Key] = body
    if input.ACL != nil {
        m.lastACL = *input.ACL
    }
    
    return &s3.PutObjectOutput{}, nil
}
```

### 7.3 並行処理のテスト

```go
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
            DiskInfoProvider: mockProvider,
        },
    }
    
    session, err := NewLocalBackupSession(config)
    require.NoError(t, err)
    defer session.Close()
    
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
```

### 7.4 エラーケースのテスト

```go
func TestErrorCases(t *testing.T) {
    t.Run("InvalidRootDir", func(t *testing.T) {
        config := LocalBackupSessionConfig{
            RootDir: "/non/existent/path",
            // その他の設定
        }
        
        _, err := NewLocalBackupSession(config)
        require.Error(t, err)
    })
    
    t.Run("WritePermissionError", func(t *testing.T) {
        // 書き込み権限のないディレクトリでテスト
        readOnlyDir := t.TempDir()
        os.Chmod(readOnlyDir, 0555)
        defer os.Chmod(readOnlyDir, 0755)
        
        config := LocalBackupSessionConfig{
            RootDir: readOnlyDir,
            // その他の設定
        }
        
        session, err := NewLocalBackupSession(config)
        require.NoError(t, err) // 作成は成功
        defer session.Close()
        
        // 保存は失敗
        err = session.Save("/tmp/test.txt", "test.txt")
        require.Error(t, err)
    })
    
    t.Run("S3NetworkError", func(t *testing.T) {
        // ネットワークエラーをシミュレート
        mockS3 := &MockS3Client{
            shouldFail: true,
            failError:  fmt.Errorf("network timeout"),
        }
        
        session := &S3BackupSession{
            config:   S3BackupSessionConfig{Bucket: "test"},
            s3Client: mockS3,
        }
        
        err := session.Save("/tmp/test.txt", "test.txt")
        require.Error(t, err)
        require.Contains(t, err.Error(), "network timeout")
    })
}
```

## 8. 結合テスト

### 8.1 MinIOを使用したS3結合テスト

dockertestを使用してMinIOコンテナを起動し、実際のS3互換環境でテストを実行します。

```go
package safebackup_test

import (
    "testing"
    "github.com/ory/dockertest/v3"
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/credentials"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/s3"
)

func TestS3BackupIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    // DockerでMinIOを起動
    pool, err := dockertest.NewPool("")
    require.NoError(t, err)
    
    resource, err := pool.RunWithOptions(&dockertest.RunOptions{
        Repository: "minio/minio",
        Tag:        "latest",
        Cmd:        []string{"server", "/data"},
        Env: []string{
            "MINIO_ROOT_USER=minioadmin",
            "MINIO_ROOT_PASSWORD=minioadmin",
        },
    })
    require.NoError(t, err)
    
    defer func() {
        if err := pool.Purge(resource); err != nil {
            t.Logf("Could not purge resource: %s", err)
        }
    }()
    
    // MinIOが起動するまで待機
    minioEndpoint := fmt.Sprintf("localhost:%s", resource.GetPort("9000/tcp"))
    
    var s3Client *s3.S3
    err = pool.Retry(func() error {
        sess, err := session.NewSession(&aws.Config{
            Endpoint:         aws.String(minioEndpoint),
            Region:           aws.String("us-east-1"),
            Credentials:      credentials.NewStaticCredentials("minioadmin", "minioadmin", ""),
            S3ForcePathStyle: aws.Bool(true),
        })
        if err != nil {
            return err
        }
        s3Client = s3.New(sess)
        
        // 接続確認
        _, err = s3Client.ListBuckets(&s3.ListBucketsInput{})
        return err
    })
    require.NoError(t, err)
    
    // テスト用バケット作成
    bucketName := "test-backup-bucket"
    _, err = s3Client.CreateBucket(&s3.CreateBucketInput{
        Bucket: aws.String(bucketName),
    })
    require.NoError(t, err)
    
    // S3BackupSessionのテスト
    config := S3BackupSessionConfig{
        Region:          "us-east-1",
        AccessKeyID:     "minioadmin",
        SecretAccessKey: "minioadmin",
        Bucket:          bucketName,
        Prefix:          "backup/",
        Endpoint:        minioEndpoint,
        ACL:             "private",
    }
    
    session, err := NewS3BackupSession(config)
    require.NoError(t, err)
    defer session.Close()
    
    // 実際のファイルアップロードテスト
    testFile := createLargeTestFile(t, 10*1024*1024) // 10MB
    err = session.Save(testFile, "integration/test.dat")
    require.NoError(t, err)
    
    // アップロードされたファイルの確認
    listResp, err := s3Client.ListObjectsV2(&s3.ListObjectsV2Input{
        Bucket: aws.String(bucketName),
        Prefix: aws.String("backup/integration/"),
    })
    require.NoError(t, err)
    require.Len(t, listResp.Contents, 1)
    require.Equal(t, "backup/integration/test.dat", *listResp.Contents[0].Key)
    require.Equal(t, int64(10*1024*1024), *listResp.Contents[0].Size)
}
```

## 9. CI/CD設定

### 9.1 GitHub Actions

`.github/workflows/ci.yml`:

```yaml
name: CI

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    strategy:
      matrix:
        os: [ubuntu-latest, windows-latest, macos-latest]
        go-version: ['1.21', '1.22']
    
    runs-on: ${{ matrix.os }}
    
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: ${{ matrix.go-version }}
    
    - name: Cache Go modules
      uses: actions/cache@v3
      with:
        path: ~/go/pkg/mod
        key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
        restore-keys: |
          ${{ runner.os }}-go-
    
    - name: Install dependencies
      run: go mod download
    
    - name: Run linter
      uses: golangci/golangci-lint-action@v3
      with:
        version: latest
    
    - name: Run unit tests
      run: go test -v -short -race -coverprofile=coverage.out ./...
    
    - name: Run integration tests (Linux only)
      if: matrix.os == 'ubuntu-latest'
      run: |
        # Docker is required for integration tests
        go test -v -race ./... -run Integration
    
    - name: Upload coverage
      if: matrix.os == 'ubuntu-latest' && matrix.go-version == '1.22'
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out
```

### 9.2 golangci-lint設定

`.golangci.yml`:

```yaml
linters:
  enable:
    - gofmt
    - govet
    - errcheck
    - staticcheck
    - ineffassign
    - misspell
    - gocritic
    - gosec

linters-settings:
  gofmt:
    simplify: true
  govet:
    check-shadowing: true
  gocritic:
    enabled-tags:
      - diagnostic
      - performance
      - style

issues:
  exclude-use-default: false
  max-issues-per-linter: 0
  max-same-issues: 0
```
