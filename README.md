# go-safe-backup

[![CI](https://github.com/ideamans/go-safe-backup/workflows/CI/badge.svg)](https://github.com/ideamans/go-safe-backup/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/ideamans/go-safe-backup.svg)](https://pkg.go.dev/github.com/ideamans/go-safe-backup)
[![License](https://img.shields.io/github/license/ideamans/go-safe-backup.svg)](LICENSE)

A capacity-safe backup library for Go that provides automatic disk space management for local backups and supports both local filesystem and Amazon S3 as backup destinations.

## Features

- **Automatic Disk Space Management**: Monitors disk usage and automatically cleans old backups when space is low
- **Session-based Architecture**: Clean API with lifecycle management
- **Multiple Backends**: Support for local filesystem and S3-compatible storage
- **Concurrent Operations**: Thread-safe operations with configurable concurrency
- **Comprehensive Testing**: Unit tests, integration tests, and mock providers

## Installation

```bash
go get github.com/ideamans/go-safe-backup
```

## Quick Start

### Local Backup

```go
package main

import (
    "context"
    "time"
    
    "github.com/ideamans/go-safe-backup"
    cleaner "github.com/ideamans/go-backup-cleaner"
)

func main() {
    config := safebackup.LocalBackupSessionConfig{
        RootDir:            "/path/to/backup/directory",
        FreeSpaceThreshold: 10 * 1024 * 1024 * 1024, // 10GB
        TargetFreeSpace:    20 * 1024 * 1024 * 1024, // 20GB
        CheckInterval:      1024 * 1024 * 1024,       // 1GB
        CleaningConfig: cleaner.CleaningConfig{
            RemoveEmptyDirs: true,
        },
    }
    
    session, err := safebackup.NewLocalBackupSession(config)
    if err != nil {
        panic(err)
    }
    defer session.Close()
    
    // Backup a file
    err = session.Save("/path/to/source/file.txt", "backups/file.txt")
    if err != nil {
        panic(err)
    }
    
    // Wait for completion
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
    defer cancel()
    
    err = session.WaitForCompletion(ctx)
    if err != nil {
        panic(err)
    }
}
```

### S3 Backup

```go
package main

import (
    "context"
    "time"
    
    "github.com/ideamans/go-safe-backup"
)

func main() {
    config := safebackup.S3BackupSessionConfig{
        Region:          "us-east-1",
        AccessKeyID:     "your-access-key",
        SecretAccessKey: "your-secret-key",
        Bucket:          "your-backup-bucket",
        Prefix:          "backups/",
        ACL:             "private",
    }
    
    session, err := safebackup.NewS3BackupSession(config)
    if err != nil {
        panic(err)
    }
    defer session.Close()
    
    // Backup a file
    err = session.Save("/path/to/source/file.txt", "2024/01/file.txt")
    if err != nil {
        panic(err)
    }
    
    // Wait for completion
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
    defer cancel()
    
    err = session.WaitForCompletion(ctx)
    if err != nil {
        panic(err)
    }
}
```

## API Reference

### BackupSession Interface

```go
type BackupSession interface {
    // Save backs up a file to the destination
    Save(localFilePath, relativePath string) error
    
    // WaitForCompletion waits for all backup and cleaning operations to complete
    WaitForCompletion(ctx context.Context) error
    
    // Close cleans up resources
    Close() error
}
```

### Local Backup Configuration

```go
type LocalBackupSessionConfig struct {
    // RootDir is the backup root directory
    RootDir string
    
    // FreeSpaceThreshold is the minimum free space before triggering cleanup (bytes)
    FreeSpaceThreshold uint64
    
    // TargetFreeSpace is the target free space after cleanup (bytes)
    TargetFreeSpace uint64
    
    // CheckInterval is the file size accumulation interval for space checks (default: 1GB)
    CheckInterval uint64
    
    // CleaningConfig is the go-backup-cleaner configuration
    CleaningConfig cleaner.CleaningConfig
}
```

### S3 Backup Configuration

```go
type S3BackupSessionConfig struct {
    // AWS credentials
    Region          string
    AccessKeyID     string
    SecretAccessKey string
    SessionToken    string // Optional
    
    // S3 settings
    Bucket   string
    Prefix   string // Optional backup prefix
    Endpoint string // Optional custom endpoint (for MinIO, etc.)
    
    // ACL settings
    ACL string // Default: private
}
```

## Development

### Prerequisites

- Go 1.22 or later
- Docker (for integration tests)

### Development Commands

```bash
# Run all tests
go test -v ./...

# Run unit tests only (skip integration tests)
go test -v -short ./...

# Run tests with race detection
go test -v -race ./...

# Run integration tests (requires Docker)
go test -v -tags=integration ./...

# Generate test coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Format code
go fmt ./...

# Run static analysis
go vet ./...

# Clean up dependencies
go mod tidy
```

### Testing with MinIO

The integration tests automatically spin up a MinIO container using Docker to test S3 functionality:

```bash
# Run integration tests
go test -v -tags=integration -run Integration ./...
```

### Project Structure

```
.
├── session.go          # Common interface definitions
├── local.go           # Local filesystem implementation
├── s3.go              # S3/MinIO implementation
├── types.go           # Shared type definitions
├── errors.go          # Error type definitions
├── local_test.go      # Local backup tests
├── s3_test.go         # S3 backup tests
├── integration_test.go # Integration tests with Docker
└── examples/          # Usage examples
    ├── local/
    └── s3/
```

## Architecture

### Session-based Design

The library uses a session-based architecture where each backup operation creates a session that manages its lifecycle:

1. **Session Creation**: Validates configuration and sets up resources
2. **File Operations**: Thread-safe backup operations with automatic space monitoring
3. **Cleanup Management**: Automatic triggering of old backup cleanup when space is low
4. **Resource Cleanup**: Proper cleanup of resources when the session is closed

### Local Backup Features

- **Automatic Space Monitoring**: Checks disk space at session creation and during operations
- **Cumulative Size Tracking**: Uses atomic operations for thread-safe size tracking
- **Configurable Thresholds**: Separate thresholds for triggering cleanup vs. target free space
- **Integration with go-backup-cleaner**: Leverages the go-backup-cleaner library for intelligent cleanup

### S3 Backup Features

- **Concurrent Uploads**: Asynchronous file uploads with configurable ACLs
- **Custom Endpoints**: Support for MinIO and other S3-compatible services
- **AWS SDK Integration**: Full compatibility with AWS S3 and IAM roles

## Error Handling

The library defines specific error types for different scenarios:

```go
var (
    ErrInvalidConfig   = errors.New("invalid configuration")
    ErrBackupFailed    = errors.New("backup failed")
    ErrCleaningTimeout = errors.New("cleaning timeout")
)
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

[Add your license information here]

## Dependencies

- [github.com/ideamans/go-backup-cleaner](https://github.com/ideamans/go-backup-cleaner) - Automatic disk space management
- [github.com/aws/aws-sdk-go](https://github.com/aws/aws-sdk-go) - AWS S3 operations
- [github.com/stretchr/testify](https://github.com/stretchr/testify) - Testing framework
- [github.com/ory/dockertest/v3](https://github.com/ory/dockertest/v3) - Integration testing with containers