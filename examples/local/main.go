package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/ideamans/go-safe-backup"
	cleaner "github.com/ideamans/go-backup-cleaner"
)

func main() {
	// Create a temporary directory for this example
	tempDir, err := os.MkdirTemp("", "backup_example_")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	fmt.Printf("Using backup directory: %s\n", tempDir)

	// Configure local backup session
	config := safebackup.LocalBackupSessionConfig{
		RootDir:            tempDir,
		FreeSpaceThreshold: 100 * 1024 * 1024,    // 100MB - low threshold for demo
		TargetFreeSpace:    200 * 1024 * 1024,    // 200MB - target after cleanup
		CheckInterval:      10 * 1024 * 1024,     // 10MB - check every 10MB
		CleaningConfig: cleaner.CleaningConfig{
			DiskInfo:        &cleaner.DefaultDiskInfoProvider{}, // Use default disk info provider
			RemoveEmptyDirs: true,
			Callbacks: cleaner.Callbacks{
				OnFileDeleted: func(info cleaner.FileDeletedInfo) {
					fmt.Printf("Cleaned up old backup: %s\n", info.Path)
				},
			},
		},
	}

	// Create backup session
	session, err := safebackup.NewLocalBackupSession(config)
	if err != nil {
		log.Fatalf("Failed to create backup session: %v", err)
	}
	defer session.Close()

	// Create some example files to backup
	exampleFiles := createExampleFiles(tempDir)
	defer cleanupExampleFiles(exampleFiles)

	fmt.Println("\nStarting backup operations...")

	// Backup multiple files
	backupPaths := []struct {
		source   string
		relative string
	}{
		{exampleFiles[0], "documents/important.txt"},
		{exampleFiles[1], "logs/application.log"},
		{exampleFiles[2], "data/config.json"},
		{exampleFiles[3], "archives/backup.tar.gz"},
	}

	for i, bp := range backupPaths {
		fmt.Printf("Backing up file %d: %s -> %s\n", i+1, bp.source, bp.relative)
		
		err = session.Save(bp.source, bp.relative)
		if err != nil {
			log.Printf("Failed to backup %s: %v", bp.source, err)
			continue
		}
		
		// Add a small delay to see the progression
		time.Sleep(100 * time.Millisecond)
	}

	// Wait for all backup operations to complete
	fmt.Println("\nWaiting for backup operations to complete...")
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = session.WaitForCompletion(ctx)
	if err != nil {
		log.Fatalf("Backup operations failed: %v", err)
	}

	fmt.Println("All backup operations completed successfully!")

	// List backed up files
	fmt.Println("\nBacked up files:")
	err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, _ := filepath.Rel(tempDir, path)
			fmt.Printf("  %s (%d bytes)\n", relPath, info.Size())
		}
		return nil
	})
	if err != nil {
		log.Printf("Failed to list files: %v", err)
	}

	fmt.Printf("\nExample completed. Backup directory: %s\n", tempDir)
	fmt.Println("Note: The directory will be cleaned up automatically when the program exits.")
}

// createExampleFiles creates some sample files for the backup example
func createExampleFiles(baseDir string) []string {
	files := []string{}
	
	examples := []struct {
		name    string
		content string
	}{
		{"important.txt", "This is an important document with critical information.\n"},
		{"application.log", "2024-01-01 10:00:00 [INFO] Application started\n2024-01-01 10:01:00 [INFO] Processing request\n"},
		{"config.json", `{"database": {"host": "localhost", "port": 5432}, "debug": true}`},
		{"backup.tar.gz", "This simulates a compressed backup file content with some binary data: \x00\x01\x02\x03"},
	}

	for _, example := range examples {
		filePath := filepath.Join(baseDir, "temp_"+example.name)
		
		err := os.WriteFile(filePath, []byte(example.content), 0644)
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
	for _, file := range files {
		os.Remove(file)
	}
}