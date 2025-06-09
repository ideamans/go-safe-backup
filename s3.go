package safebackup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// S3API defines the interface for S3 operations used by the backup session
type S3API interface {
	HeadBucket(input *s3.HeadBucketInput) (*s3.HeadBucketOutput, error)
	PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error)
}

// S3BackupSession はS3へのバックアップセッション実装
type S3BackupSession struct {
	config   S3BackupSessionConfig
	s3Client S3API
	wg       sync.WaitGroup
}

// NewS3BackupSession はS3バックアップセッションインスタンスを作成
func NewS3BackupSession(config S3BackupSessionConfig) (*S3BackupSession, error) {
	// 設定の検証
	if err := validateS3Config(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// デフォルト値の設定
	if config.ACL == "" {
		config.ACL = "private"
	}

	// AWS設定の構築
	awsConfig := &aws.Config{
		Region: aws.String(config.Region),
	}

	// 認証情報の設定
	if config.AccessKeyID != "" && config.SecretAccessKey != "" {
		awsConfig.Credentials = credentials.NewStaticCredentials(
			config.AccessKeyID,
			config.SecretAccessKey,
			config.SessionToken,
		)
	}

	// カスタムエンドポイントの設定（MinIO等）
	if config.Endpoint != "" {
		awsConfig.Endpoint = aws.String(config.Endpoint)
		awsConfig.S3ForcePathStyle = aws.Bool(true)
	}

	// AWSセッションの作成
	sess, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	// S3クライアントの作成
	s3Client := s3.New(sess)

	// バケットの存在確認
	_, err = s3Client.HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(config.Bucket),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to access bucket %s: %w", config.Bucket, err)
	}

	return &S3BackupSession{
		config:   config,
		s3Client: s3Client,
	}, nil
}

// Save はファイルをS3にアップロードする
func (s *S3BackupSession) Save(localFilePath, relativePath string) error {
	// 入力検証
	if localFilePath == "" || relativePath == "" {
		return fmt.Errorf("%w: empty file path", ErrInvalidConfig)
	}

	// ファイルの存在確認
	fileInfo, err := os.Stat(localFilePath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	if !fileInfo.Mode().IsRegular() {
		return fmt.Errorf("%w: source is not a regular file", ErrInvalidConfig)
	}

	// S3キーの構築
	key := filepath.ToSlash(filepath.Join(s.config.Prefix, relativePath))

	// S3にアップロード
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		_ = s.uploadFile(localFilePath, key, fileInfo.Size())
	}()

	return nil
}

// uploadFile は実際のアップロード処理を行う
func (s *S3BackupSession) uploadFile(filePath, key string, size int64) error {
	// ファイルを開く
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	input := &s3.PutObjectInput{
		Bucket:        aws.String(s.config.Bucket),
		Key:           aws.String(key),
		Body:          file,
		ContentLength: aws.Int64(size),
	}

	// ACLの設定
	if s.config.ACL != "" {
		input.ACL = aws.String(s.config.ACL)
	}

	// アップロード実行
	_, err = s.s3Client.PutObject(input)
	if err != nil {
		return fmt.Errorf("%w: failed to upload to S3: %v", ErrBackupFailed, err)
	}

	return nil
}

// WaitForCompletion はすべてのアップロードの完了を待つ
func (s *S3BackupSession) WaitForCompletion(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("upload timeout: %w", ctx.Err())
	case <-done:
		return nil
	}
}

// Close はリソースをクリーンアップする
func (s *S3BackupSession) Close() error {
	// S3クライアントは特にクリーンアップ不要
	return nil
}

// validateS3Config はS3バックアップ設定を検証する
func validateS3Config(config S3BackupSessionConfig) error {
	if config.Region == "" {
		return fmt.Errorf("%w: region is required", ErrInvalidConfig)
	}

	if config.Bucket == "" {
		return fmt.Errorf("%w: bucket name is required", ErrInvalidConfig)
	}

	// 認証情報のチェック（エンドポイントがない場合は必須）
	// IAMロールやEC2インスタンスプロファイルを使用する場合もあるため、チェックのみ
	_ = config.Endpoint == "" && (config.AccessKeyID == "" || config.SecretAccessKey == "")

	// ACLの妥当性チェック
	validACLs := map[string]bool{
		"private":                   true,
		"public-read":               true,
		"public-read-write":         true,
		"authenticated-read":        true,
		"aws-exec-read":             true,
		"bucket-owner-read":         true,
		"bucket-owner-full-control": true,
		"":                          true, // デフォルト値
	}

	if !validACLs[config.ACL] {
		return fmt.Errorf("%w: invalid ACL value: %s", ErrInvalidConfig, config.ACL)
	}

	return nil
}
