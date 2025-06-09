module github.com/ideamans/go-safe-backup/examples/local

go 1.22.2

replace github.com/ideamans/go-safe-backup => ../..

require (
	github.com/ideamans/go-backup-cleaner v1.0.1
	github.com/ideamans/go-safe-backup v0.0.0-00010101000000-000000000000
)

require (
	github.com/aws/aws-sdk-go v1.55.7 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
)
