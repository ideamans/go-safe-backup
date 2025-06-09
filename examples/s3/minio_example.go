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

// MinIOExample demonstrates how to use go-safe-backup with MinIO
// This is useful for local development and testing
func MinIOExample() {
	fmt.Println("=== MinIO S3 Backup Example ===")
	fmt.Println("This example assumes MinIO is running locally on port 9000")
	fmt.Println("You can start MinIO with: docker run -p 9000:9000 -p 9001:9001 minio/minio server /data --console-address ':9001'")
	fmt.Println()

	// MinIO configuration for local development
	config := safebackup.S3BackupSessionConfig{
		Region:          "us-east-1", // MinIO accepts any region
		AccessKeyID:     "minioadmin", // Default MinIO credentials
		SecretAccessKey: "minioadmin", // Default MinIO credentials
		Bucket:          "test-backup-bucket",
		Prefix:          "dev-backups/",
		Endpoint:        "http://localhost:9000", // MinIO endpoint
		ACL:             "private",
	}

	fmt.Printf("Connecting to MinIO at: %s\n", config.Endpoint)
	fmt.Printf("Using bucket: %s\n", config.Bucket)

	// Create S3 backup session
	session, err := safebackup.NewS3BackupSession(config)
	if err != nil {
		log.Fatalf("Failed to connect to MinIO: %v", err)
		log.Fatalf("Make sure MinIO is running and the bucket '%s' exists", config.Bucket)
	}
	defer session.Close()

	// Create example files
	exampleFiles := createMinIOExampleFiles()
	defer cleanupExampleFiles(exampleFiles)

	fmt.Println("\nUploading files to MinIO...")

	// Upload files with timestamp-based organization
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	
	backupPaths := []struct {
		source   string
		relative string
	}{
		{exampleFiles[0], fmt.Sprintf("daily/%s/database.sql", timestamp)},
		{exampleFiles[1], fmt.Sprintf("daily/%s/config.yaml", timestamp)},
		{exampleFiles[2], fmt.Sprintf("weekly/%s/full-backup.tar.gz", timestamp)},
		{exampleFiles[3], fmt.Sprintf("logs/%s/access.log", timestamp)},
	}

	for i, bp := range backupPaths {
		fmt.Printf("  [%d/%d] %s -> %s\n", i+1, len(backupPaths), 
			filepath.Base(bp.source), bp.relative)
		
		err = session.Save(bp.source, bp.relative)
		if err != nil {
			log.Printf("Failed to upload %s: %v", bp.source, err)
			continue
		}
	}

	// Wait for completion
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	err = session.WaitForCompletion(ctx)
	if err != nil {
		log.Fatalf("Upload failed: %v", err)
	}

	fmt.Println("\n✅ All files uploaded successfully!")
	
	// Print MinIO console information
	fmt.Println("\n📊 MinIO Console Information:")
	fmt.Println("  Console URL: http://localhost:9001")
	fmt.Println("  Username: minioadmin")
	fmt.Println("  Password: minioadmin")
	fmt.Println()
	fmt.Println("🗂️  Uploaded files:")
	for _, bp := range backupPaths {
		fmt.Printf("   s3://%s/%s%s\n", config.Bucket, config.Prefix, bp.relative)
	}
	
	fmt.Println("\n💡 You can view these files in the MinIO console browser.")
}

// createMinIOExampleFiles creates sample files specifically for the MinIO example
func createMinIOExampleFiles() []string {
	tempDir, err := os.MkdirTemp("", "minio_backup_example_")
	if err != nil {
		log.Fatal(err)
	}

	files := []string{}
	
	examples := []struct {
		name    string
		content string
	}{
		{"database.sql", `-- Database backup
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO users (username, email) VALUES 
('admin', 'admin@example.com'),
('user1', 'user1@example.com');
`},
		{"config.yaml", `# Application configuration
server:
  host: 0.0.0.0
  port: 8080
  
database:
  host: localhost
  port: 5432
  name: myapp
  ssl_mode: require
  
logging:
  level: info
  format: json
`},
		{"full-backup.tar.gz", "This simulates a compressed full system backup containing:\n- Application files\n- Database dumps\n- Configuration files\n- User data\n"},
		{"access.log", `2024-01-01T10:00:00Z INFO GET /api/users 200 45ms
2024-01-01T10:00:15Z INFO POST /api/users 201 123ms
2024-01-01T10:00:30Z WARN GET /api/users/999 404 12ms
2024-01-01T10:00:45Z INFO GET /health 200 2ms
2024-01-01T10:01:00Z INFO GET /api/users 200 38ms
`},
	}

	for _, example := range examples {
		filePath := filepath.Join(tempDir, example.name)
		
		err := os.WriteFile(filePath, []byte(example.content), 0644)
		if err != nil {
			log.Printf("Failed to create example file %s: %v", filePath, err)
			continue
		}
		
		files = append(files, filePath)
	}

	return files
}

// Uncomment the function below and comment out the main function above to run the MinIO example
/*
func main() {
	MinIOExample()
}
*/