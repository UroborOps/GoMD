package s3backup

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	gomdcfg "github.com/UroborOps/GoMD/backend/internal/config"
)

// Start initializes the S3 backup worker.
func Start(cfg *gomdcfg.Config) {
	if !cfg.S3BackupEnabled {
		return
	}

	if cfg.S3Endpoint == "" || cfg.S3Bucket == "" || cfg.S3AccessKey == "" || cfg.S3SecretKey == "" {
		log.Println("s3backup: enabled but missing required configuration (endpoint, bucket, access_key, secret_key).")
		return
	}

	interval := cfg.S3BackupInterval
	if interval <= 0 {
		interval = 60 // Default to 60 minutes
	}

	retainCount := cfg.S3BackupRetainCount
	if retainCount <= 0 {
		retainCount = 7 // Default to retaining 7 backups
	}

	log.Printf("s3backup: initializing backup worker for %s (interval: %dm, retain: %d)", cfg.VaultPath, interval, retainCount)

	// Do initial backup
	performBackup(cfg, retainCount)

	go func() {
		ticker := time.NewTicker(time.Duration(interval) * time.Minute)
		for range ticker.C {
			performBackup(cfg, retainCount)
		}
	}()
}

func performBackup(cfg *gomdcfg.Config, retainCount int) {
	ctx := context.Background()

	// Configure AWS SDK v2
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               cfg.S3Endpoint,
			HostnameImmutable: true,
		}, nil
	})

	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.S3Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.S3AccessKey, cfg.S3SecretKey, "")),
		config.WithEndpointResolverWithOptions(customResolver),
	)
	if err != nil {
		log.Printf("s3backup error: failed to load aws config: %v", err)
		return
	}

	s3Client := s3.NewFromConfig(awsCfg)

	// Create zip file
	zipFile, err := os.CreateTemp("", "gomd-vault-*.zip")
	if err != nil {
		log.Printf("s3backup error: failed to create temp file: %v", err)
		return
	}
	tempPath := zipFile.Name()
	defer os.Remove(tempPath)

	err = zipVault(cfg.VaultPath, zipFile)
	zipFile.Close() // Close to flush writes before uploading
	if err != nil {
		log.Printf("s3backup error: failed to zip vault: %v", err)
		return
	}

	// Re-open for reading
	fileToUpload, err := os.Open(tempPath)
	if err != nil {
		log.Printf("s3backup error: failed to open temp zip: %v", err)
		return
	}
	defer fileToUpload.Close()

	// Upload to S3
	timestamp := time.Now().UTC().Format("20060102T150405Z")
	objectName := fmt.Sprintf("vault-backup-%s.zip", timestamp)

	contentType := "application/zip"
	_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(cfg.S3Bucket),
		Key:         aws.String(objectName),
		Body:        fileToUpload,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		log.Printf("s3backup error: failed to upload backup: %v", err)
		return
	}

	log.Printf("s3backup: successfully uploaded backup %s", objectName)

	// Prune old backups
	pruneBackups(ctx, s3Client, cfg.S3Bucket, retainCount)
}

func pruneBackups(ctx context.Context, client *s3.Client, bucket string, retainCount int) {
	prefix := "vault-backup-"
	listOutput, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		log.Printf("s3backup warning: error listing objects for pruning: %v", err)
		return
	}

	objects := listOutput.Contents
	if len(objects) <= retainCount {
		return
	}

	// Sort by LastModified, oldest first
	sort.Slice(objects, func(i, j int) bool {
		if objects[i].LastModified == nil || objects[j].LastModified == nil {
			return false
		}
		return objects[i].LastModified.Before(*objects[j].LastModified)
	})

	// Delete oldest objects to keep only retainCount
	toDelete := len(objects) - retainCount
	var deleteObjects []types.ObjectIdentifier
	for i := 0; i < toDelete; i++ {
		deleteObjects = append(deleteObjects, types.ObjectIdentifier{
			Key: objects[i].Key,
		})
		log.Printf("s3backup: queueing old backup %s for deletion", *objects[i].Key)
	}

	_, err = client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(bucket),
		Delete: &types.Delete{
			Objects: deleteObjects,
			Quiet:   aws.Bool(true),
		},
	})
	if err != nil {
		log.Printf("s3backup warning: failed to delete old backups: %v", err)
	} else {
		log.Printf("s3backup: successfully pruned %d old backups", toDelete)
	}
}

func zipVault(source string, target io.Writer) error {
	zipWriter := zip.NewWriter(target)
	defer zipWriter.Close()

	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .git directory if present
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		// Skip the root directory itself, just add its contents
		if path == source {
			return nil
		}

		// Create header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}

		header.Name = relPath

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})
}
