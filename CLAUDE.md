# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

go-safe-backup (safebackup) is a capacity-safe backup library for Go that provides automatic disk space management for local backups and supports both local filesystem and Amazon S3 as backup destinations.

## Key Architecture

### Core Interface
The library uses a session-based architecture with a common `BackupSession` interface:
- `Save(localFilePath, relativePath string) error` - Backs up a file
- `WaitForCompletion(ctx context.Context) error` - Waits for all operations to complete
- `Close() error` - Cleans up resources

### Local Backup Implementation
- Automatically checks disk space at session creation and during operation
- Uses `go-backup-cleaner` to delete old backups when space is low
- Implements cumulative size tracking with configurable check intervals
- Performs initial cleaning when creating new sessions if disk space is below threshold

### S3 Backup Implementation
- Uses AWS SDK for Go
- Supports custom endpoints (MinIO compatible)
- Handles ACL configuration

## Common Development Commands

### Build and Test
```bash
# Run all tests
go test -v ./...

# Run unit tests only (skip integration tests)
go test -v -short ./...

# Run tests with race detection
go test -v -race ./...

# Run integration tests (requires Docker)
go test -v -run Integration ./...

# Generate test coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Linting and Formatting
```bash
# Format code
go fmt ./...

# Run go vet
go vet ./...

# Run golangci-lint (if installed)
golangci-lint run
```

### Dependency Management
```bash
# Download dependencies
go mod download

# Add missing dependencies and remove unused ones
go mod tidy

# Verify dependencies
go mod verify
```

## Project Structure

The library follows this package structure:
- `session.go` - Common interface definitions
- `local.go` - Local filesystem implementation with automatic space management
- `s3.go` - S3/MinIO implementation
- `types.go` - Shared type definitions
- `errors.go` - Error type definitions
- Test files follow Go convention: `*_test.go`

## Testing Approach

### Unit Tests
- Mock providers for disk info and S3 client
- Test concurrent operations and edge cases
- Use `testify/require` for assertions

### Integration Tests
- Use `dockertest` to spin up MinIO containers for S3 testing
- Skip with `-short` flag for faster development cycles
- Test actual file operations and S3 uploads

## Key Dependencies

- `github.com/ideamans/go-backup-cleaner` - For automatic disk space management (see [GitHub repository](https://github.com/ideamans/go-backup-cleaner) for documentation)
- `github.com/aws/aws-sdk-go` - For S3 operations
- `github.com/stretchr/testify` - For test assertions
- `github.com/ory/dockertest/v3` - For integration testing with containers

## Important Design Decisions

1. **Session-based API**: Each backup operation creates a session that manages its lifecycle
2. **Automatic initial cleaning**: Local sessions check and clean disk space immediately upon creation
3. **Atomic size tracking**: Uses atomic operations for thread-safe cumulative size tracking
4. **Configurable thresholds**: Separate thresholds for triggering cleaning vs. target free space
5. **Context support**: All blocking operations support context for cancellation/timeout

## Examples and Usage

The project includes comprehensive examples in the `examples/` directory:

### Local Backup Example (`examples/local/`)
- Demonstrates local filesystem backup with automatic disk space management
- Shows integration with go-backup-cleaner for automatic cleanup
- Includes practical file organization and monitoring
- Run with: `cd examples/local && go run main.go`

### S3 Backup Example (`examples/s3/`)
- Shows AWS S3 and MinIO integration
- Demonstrates concurrent file uploads and organized storage
- Includes environment variable configuration
- Run with proper AWS credentials: `cd examples/s3 && go run main.go`
- MinIO example available in `minio_example.go` for local testing

### Example Features
- Real-world file backup scenarios
- Error handling and progress monitoring
- Configuration via environment variables
- Cleanup and resource management
- Integration patterns for databases, logs, and configuration files

## Development Guidelines

- When investigating library dependencies or source code, limit exploration to this project directory and below
- Do not explore parent directories or unrelated codebases when analyzing this project
- Refer to examples in `examples/` directory for practical usage patterns
- Test examples regularly to ensure they work with API changes
