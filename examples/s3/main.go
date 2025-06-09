package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/ideamans/go-safe-backup"
)

func main() {
	// S3 configuration - you can modify these values or use environment variables
	config := safebackup.S3BackupSessionConfig{
		Region:          getEnvOrDefault("AWS_REGION", "us-east-1"),
		AccessKeyID:     getEnvOrDefault("AWS_ACCESS_KEY_ID", ""),
		SecretAccessKey: getEnvOrDefault("AWS_SECRET_ACCESS_KEY", ""),
		SessionToken:    os.Getenv("AWS_SESSION_TOKEN"), // Optional
		Bucket:          getEnvOrDefault("S3_BUCKET", "my-backup-bucket"),
		Prefix:          getEnvOrDefault("S3_PREFIX", "backups/"),
		Endpoint:        os.Getenv("S3_ENDPOINT"), // Optional, for MinIO or other S3-compatible services
		ACL:             getEnvOrDefault("S3_ACL", "private"),
	}

	// Validate required configuration
	if config.Bucket == "" {
		log.Fatal("S3_BUCKET environment variable or bucket name must be specified")
	}

	// For AWS, credentials can also be provided via IAM roles, instance profiles, or credential files
	if config.AccessKeyID == "" && config.Endpoint != "" {
		log.Fatal("AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY must be provided for custom endpoints")
	}

	fmt.Printf("Backing up to S3 bucket: %s\n", config.Bucket)
	if config.Endpoint != "" {
		fmt.Printf("Using custom endpoint: %s\n", config.Endpoint)
	}

	// Create S3 backup session
	session, err := safebackup.NewS3BackupSession(config)
	if err != nil {
		log.Fatalf("Failed to create S3 backup session: %v", err)
	}
	defer session.Close()

	// Create some example files to backup
	exampleFiles := createExampleFiles()
	defer cleanupExampleFiles(exampleFiles)

	fmt.Println("\nStarting S3 backup operations...")

	// Backup multiple files with organized structure
	backupPaths := []struct {
		source   string
		relative string
	}{
		{exampleFiles[0], "documents/2024/01/important.txt"},
		{exampleFiles[1], "logs/2024/01/application.log"},
		{exampleFiles[2], "config/app/config.json"},
		{exampleFiles[3], "archives/2024/01/backup.tar.gz"},
		{exampleFiles[4], "images/profile/avatar.jpg"},
	}

	for i, bp := range backupPaths {
		fmt.Printf("Uploading file %d: %s -> %s%s\n", i+1, 
			filepath.Base(bp.source), config.Prefix, bp.relative)
		
		err = session.Save(bp.source, bp.relative)
		if err != nil {
			log.Printf("Failed to upload %s: %v", bp.source, err)
			continue
		}
	}

	// Wait for all upload operations to complete
	fmt.Println("\nWaiting for all uploads to complete...")
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err = session.WaitForCompletion(ctx)
	if err != nil {
		log.Fatalf("Upload operations failed: %v", err)
	}

	fmt.Println("All files uploaded successfully to S3!")
	
	// Print summary
	fmt.Printf("\nBackup Summary:\n")
	fmt.Printf("- Bucket: %s\n", config.Bucket)
	fmt.Printf("- Prefix: %s\n", config.Prefix)
	fmt.Printf("- Files uploaded: %d\n", len(backupPaths))
	fmt.Printf("- ACL: %s\n", config.ACL)

	fmt.Println("\nFiles uploaded:")
	for _, bp := range backupPaths {
		fmt.Printf("  s3://%s/%s%s\n", config.Bucket, config.Prefix, bp.relative)
	}

	fmt.Println("\nExample completed successfully!")
}

// createExampleFiles creates some sample files for the S3 backup example
func createExampleFiles() []string {
	tempDir, err := os.MkdirTemp("", "s3_backup_example_")
	if err != nil {
		log.Fatal(err)
	}

	files := []string{}
	
	examples := []struct {
		name    string
		content string
		size    int // for binary files
	}{
		{"important.txt", "This is an important document that needs to be backed up to S3.\nIt contains critical business information.\n", 0},
		{"application.log", "2024-01-01 10:00:00 [INFO] Application started\n2024-01-01 10:01:00 [INFO] Processing request\n2024-01-01 10:02:00 [INFO] Request completed\n", 0},
		{"config.json", `{
  "database": {
    "host": "localhost",
    "port": 5432,
    "ssl": true
  },
  "api": {
    "version": "v1",
    "timeout": 30
  },
  "debug": false
}`, 0},
		{"backup.tar.gz", "This simulates a compressed backup file", 0},
		{"avatar.jpg", "", 1024}, // Binary file simulation
	}

	for _, example := range examples {
		filePath := filepath.Join(tempDir, example.name)
		
		var content []byte
		if example.size > 0 {
			// Create binary content for files like images
			content = make([]byte, example.size)
			for i := range content {
				content[i] = byte(i % 256)
			}
		} else {
			content = []byte(example.content)
		}
		
		err := os.WriteFile(filePath, content, 0644)
		if err != nil {
			log.Printf("Failed to create example file %s: %v", filePath, err)
			continue
		}
		
		files = append(files, filePath)
	}

	return files
}

// cleanupExampleFiles removes the temporary example files
func cleanupExampleFiles(files []string) {
	if len(files) == 0 {
		return
	}
	
	// Remove the temporary directory
	tempDir := filepath.Dir(files[0])
	os.RemoveAll(tempDir)
}

// getEnvOrDefault returns the environment variable value or a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}