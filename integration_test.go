//go:build integration
// +build integration

package safebackup

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/require"
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
			Endpoint:         aws.String("http://" + minioEndpoint),
			Region:           aws.String("us-east-1"),
			Credentials:      credentials.NewStaticCredentials("minioadmin", "minioadmin", ""),
			S3ForcePathStyle: aws.Bool(true),
			DisableSSL:       aws.Bool(true),
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
		Endpoint:        "http://" + minioEndpoint,
		ACL:             "private",
	}

	session, err := NewS3BackupSession(config)
	require.NoError(t, err)
	defer session.Close()

	// 実際のファイルアップロードテスト
	testFile := createLargeTestFile(t, 10*1024*1024) // 10MB
	err = session.Save(testFile, "integration/test.dat")
	require.NoError(t, err)

	// アップロード完了を待つ
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	err = session.WaitForCompletion(ctx)
	cancel()
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

func TestS3BackupIntegrationMultipleFiles(t *testing.T) {
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
			Endpoint:         aws.String("http://" + minioEndpoint),
			Region:           aws.String("us-east-1"),
			Credentials:      credentials.NewStaticCredentials("minioadmin", "minioadmin", ""),
			S3ForcePathStyle: aws.Bool(true),
			DisableSSL:       aws.Bool(true),
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
	bucketName := "test-multi-backup-bucket"
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
		Prefix:          "multi-backup/",
		Endpoint:        "http://" + minioEndpoint,
		ACL:             "private",
	}

	session, err := NewS3BackupSession(config)
	require.NoError(t, err)
	defer session.Close()

	// 複数のファイルをアップロード
	files := []struct {
		size         int64
		relativePath string
	}{
		{5 * 1024 * 1024, "2024/01/file1.dat"},     // 5MB
		{3 * 1024 * 1024, "2024/01/file2.dat"},     // 3MB
		{1 * 1024 * 1024, "2024/02/file3.dat"},     // 1MB
		{2 * 1024 * 1024, "2024/02/sub/file4.dat"}, // 2MB
	}

	for _, f := range files {
		testFile := createLargeTestFile(t, f.size)
		err := session.Save(testFile, f.relativePath)
		require.NoError(t, err)
	}

	// アップロード完了を待つ
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	err = session.WaitForCompletion(ctx)
	cancel()
	require.NoError(t, err)

	// アップロードされたファイルの確認
	listResp, err := s3Client.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
		Prefix: aws.String("multi-backup/"),
	})
	require.NoError(t, err)
	require.Len(t, listResp.Contents, 4)

	// 各ファイルのサイズを確認
	expectedFiles := map[string]int64{
		"multi-backup/2024/01/file1.dat":     5 * 1024 * 1024,
		"multi-backup/2024/01/file2.dat":     3 * 1024 * 1024,
		"multi-backup/2024/02/file3.dat":     1 * 1024 * 1024,
		"multi-backup/2024/02/sub/file4.dat": 2 * 1024 * 1024,
	}

	for _, obj := range listResp.Contents {
		expectedSize, ok := expectedFiles[*obj.Key]
		require.True(t, ok, "Unexpected file: %s", *obj.Key)
		require.Equal(t, expectedSize, *obj.Size, "Size mismatch for %s", *obj.Key)
	}
}

// createLargeTestFile は指定サイズの大きなテストファイルを作成
func createLargeTestFile(t *testing.T, size int64) string {
	file, err := os.CreateTemp("", "large_test_")
	require.NoError(t, err)

	// バッファサイズを設定（1MB）
	bufferSize := int64(1024 * 1024)
	buffer := make([]byte, bufferSize)

	// ランダムデータで埋める
	for i := range buffer {
		buffer[i] = byte(i % 256)
	}

	// 指定サイズまでデータを書き込む
	written := int64(0)
	for written < size {
		toWrite := bufferSize
		if size-written < bufferSize {
			toWrite = size - written
		}
		n, err := file.Write(buffer[:toWrite])
		require.NoError(t, err)
		written += int64(n)
	}

	err = file.Close()
	require.NoError(t, err)

	t.Cleanup(func() {
		os.Remove(file.Name())
	})

	return file.Name()
}
