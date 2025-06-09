# Examples

This directory contains practical examples of how to use the go-safe-backup library.

## Local Backup Example

The local backup example demonstrates:
- Setting up a local backup session with disk space management
- Backing up multiple files with organized directory structure
- Automatic cleanup of old backups when disk space is low
- Monitoring backup progress and completion

### Running the Local Example

```bash
cd examples/local
go run main.go
```

This example will:
1. Create a temporary backup directory
2. Generate sample files to backup
3. Configure automatic cleanup when space is low
4. Backup files to an organized directory structure
5. Show the backed up files
6. Clean up automatically when finished

## S3 Backup Example

The S3 backup example demonstrates:
- Setting up an S3 backup session
- Uploading files to AWS S3 or S3-compatible storage
- Organizing files with prefixes and timestamps
- Handling concurrent uploads

### Running the S3 Example

#### With AWS S3

Set your AWS credentials and run:

```bash
export AWS_REGION="us-east-1"
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export S3_BUCKET="your-backup-bucket"
export S3_PREFIX="backups/"

cd examples/s3
go run main.go
```

#### With MinIO (Local S3-compatible storage)

1. Start MinIO:
```bash
docker run -p 9000:9000 -p 9001:9001 \
  -e "MINIO_ROOT_USER=minioadmin" \
  -e "MINIO_ROOT_PASSWORD=minioadmin" \
  minio/minio server /data --console-address ":9001"
```

2. Create a bucket in MinIO console (http://localhost:9001) or using mc client:
```bash
mc alias set local http://localhost:9000 minioadmin minioadmin
mc mb local/test-backup-bucket
```

3. Run the MinIO example:
```bash
cd examples/s3
# Edit minio_example.go to uncomment the main function
go run minio_example.go
```

### Environment Variables

The S3 example supports these environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `AWS_REGION` | `us-east-1` | AWS region |
| `AWS_ACCESS_KEY_ID` | | AWS access key (required for custom endpoints) |
| `AWS_SECRET_ACCESS_KEY` | | AWS secret key (required for custom endpoints) |
| `AWS_SESSION_TOKEN` | | AWS session token (optional) |
| `S3_BUCKET` | `my-backup-bucket` | S3 bucket name |
| `S3_PREFIX` | `backups/` | Prefix for all uploaded files |
| `S3_ENDPOINT` | | Custom S3 endpoint (for MinIO, etc.) |
| `S3_ACL` | `private` | S3 object ACL |

## Example Files Structure

### Local Example Output
```
backup_directory/
├── documents/
│   └── important.txt
├── logs/
│   └── application.log
├── data/
│   └── config.json
└── archives/
    └── backup.tar.gz
```

### S3 Example Output
```
s3://your-bucket/backups/
├── documents/2024/01/important.txt
├── logs/2024/01/application.log
├── config/app/config.json
├── archives/2024/01/backup.tar.gz
└── images/profile/avatar.jpg
```

## Integration with Real Applications

These examples can be adapted for real-world use cases:

### Database Backups
```go
// Backup database dump
err = session.Save("/tmp/database_dump.sql", 
    fmt.Sprintf("database/daily/%s.sql", time.Now().Format("2006-01-02")))
```

### Log Rotation
```go
// Backup rotated logs
err = session.Save("/var/log/app.log.1", 
    fmt.Sprintf("logs/%s/app.log", time.Now().Format("2006-01")))
```

### Configuration Backups
```go
// Backup configuration files
err = session.Save("/etc/myapp/config.yaml", 
    fmt.Sprintf("config/%s/config.yaml", hostname))
```

## Best Practices

1. **Use appropriate prefixes** to organize your backups
2. **Set reasonable thresholds** for disk space management
3. **Use context timeouts** for long-running operations
4. **Handle errors gracefully** and implement retry logic if needed
5. **Monitor backup success** and implement alerting
6. **Test restore procedures** regularly
7. **Use IAM roles** instead of hardcoded credentials when possible