# Contributing to go-safe-backup

Thank you for your interest in contributing to go-safe-backup!

## Development Setup

1. **Clone the repository**:
   ```bash
   git clone https://github.com/ideamans/go-safe-backup.git
   cd go-safe-backup
   ```

2. **Install dependencies**:
   ```bash
   go mod download
   ```

3. **Run tests**:
   ```bash
   # Unit tests
   go test -v -short ./...
   
   # Integration tests (requires Docker)
   go test -v -tags=integration ./...
   
   # All tests with race detection
   go test -v -race ./...
   ```

## Code Guidelines

- Follow standard Go formatting (`go fmt`)
- Run `go vet` before submitting
- Write tests for new functionality
- Keep functions focused and reasonably sized

## Pull Request Process

1. Fork the repository and create your branch from `main`
2. Make your changes and add tests
3. Ensure all tests pass
4. Run `go fmt` and `go vet`
5. Submit a pull request

## Testing

- Write unit tests for new functionality
- Use table-driven tests where appropriate
- Ensure integration tests pass with MinIO

## Examples

Test your changes with the examples:

```bash
# Test local example
cd examples/local && go run main.go

# Test S3 example (with credentials)
cd examples/s3 && go run main.go
```

Thank you for contributing!