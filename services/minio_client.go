package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var minioClient *minio.Client
var returnURL string

// ensureMinioClient ensures MinIO client is initialized
func ensureMinioClient() error {
	if minioClient != nil {
		return nil
	}
	return InitMinioClient()
}

// InitMinioClient initializes MinIO client
func InitMinioClient() error {
	var err error

	// Get configuration from environment
	endpoint := os.Getenv("MINIO_ENDPOINT")
	accessKey := os.Getenv("MINIO_ACCESS_KEY_ID")
	secretKey := os.Getenv("MINIO_SECRET_ACCESS_KEY")
	useSSL := os.Getenv("MINIO_USE_SSL") == "true"
	returnURL = os.Getenv("MINIO_RETURN_URL")

	if endpoint == "" || accessKey == "" || secretKey == "" {
		return fmt.Errorf("MinIO configuration missing in environment variables")
	}

	if returnURL == "" {
		returnURL = fmt.Sprintf("http://%s", endpoint)
	}

	minioClient, err = minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return fmt.Errorf("failed to create MinIO client: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = minioClient.ListBuckets(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to MinIO: %w", err)
	}

	log.Printf("Successfully initialized MinIO client at %s", endpoint)
	return nil
}

// UploadFile uploads a file to MinIO (similar to your UploadFile function)
func UploadFile(fileBytes []byte, fileName, bucket, rootPath string) (string, error) {
	if err := ensureMinioClient(); err != nil {
		return "", fmt.Errorf("failed to initialize MinIO client: %w", err)
	}

	ctx := context.Background()

	// Use default bucket if not specified
	if bucket == "" {
		bucket = os.Getenv("MINIO_BUCKET_NAME")
		if bucket == "" {
			bucket = "osm-data"
		}
	}

	// Create bucket if it doesn't exist
	err := minioClient.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: "us-east-1"})
	if err != nil {
		// Check if bucket already exists
		exists, errBucketExists := minioClient.BucketExists(ctx, bucket)
		if errBucketExists != nil {
			return "", fmt.Errorf("failed to check bucket existence: %w", errBucketExists)
		}
		if !exists {
			return "", fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	// Generate object name with timestamp and UUID-like suffix
	dt := time.Now()
	objectName := dt.Format("20060102150405") + "_" + fmt.Sprintf("%d", time.Now().UnixNano())
	if rootPath != "" {
		objectName = rootPath + "/" + objectName
	}
	if fileName != "" {
		objectName = objectName + "_" + fileName
	}

	// Detect content type
	contentType := http.DetectContentType(fileBytes)

	log.Printf("Starting upload file %s to bucket %s", objectName, bucket)

	// Upload the file
	info, err := minioClient.PutObject(ctx, bucket, objectName, bytes.NewReader(fileBytes), int64(len(fileBytes)), minio.PutObjectOptions{
		ContentType: contentType,
		UserMetadata: map[string]string{
			"filename": fileName,
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	log.Printf("Successfully uploaded %s of size %d", objectName, info.Size)

	// Return the URL
	uploadURL := fmt.Sprintf("%s/%s/%s", returnURL, bucket, objectName)
	return uploadURL, nil
}

// UploadPolygonData uploads polygon data specifically for OSM data
func UploadPolygonData(polygonData []byte, objectName string) (string, error) {
	if err := ensureMinioClient(); err != nil {
		return "", fmt.Errorf("failed to initialize MinIO client: %w", err)
	}

	ctx := context.Background()

	bucket := os.Getenv("MINIO_BUCKET_NAME")
	if bucket == "" {
		bucket = "osm-data"
	}

	// Create bucket if it doesn't exist
	err := minioClient.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: "us-east-1"})
	if err != nil {
		// Check if bucket already exists
		exists, errBucketExists := minioClient.BucketExists(ctx, bucket)
		if errBucketExists != nil {
			return "", fmt.Errorf("failed to check bucket existence: %w", errBucketExists)
		}
		if !exists {
			return "", fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	log.Printf("Starting upload polygon data %s to bucket %s", objectName, bucket)

	// Upload the polygon data
	info, err := minioClient.PutObject(ctx, bucket, objectName, bytes.NewReader(polygonData), int64(len(polygonData)), minio.PutObjectOptions{
		ContentType: "application/json",
		UserMetadata: map[string]string{
			"type": "polygon-data",
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload polygon data: %w", err)
	}

	log.Printf("Successfully uploaded polygon data %s of size %d", objectName, info.Size)

	// Return the URL
	uploadURL := fmt.Sprintf("%s/%s/%s", returnURL, bucket, objectName)
	return uploadURL, nil
}

// DownloadFile downloads a file from MinIO
func DownloadFile(bucket, objectName string) ([]byte, error) {
	if err := ensureMinioClient(); err != nil {
		return nil, fmt.Errorf("failed to initialize MinIO client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	object, err := minioClient.GetObject(ctx, bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer object.Close()

	// Read all data
	data, err := io.ReadAll(object)
	if err != nil {
		return nil, fmt.Errorf("failed to read object data: %w", err)
	}

	return data, nil
}

// GetPresignedURL generates a presigned URL for object access
func GetPresignedURL(bucket, objectName string, expiry time.Duration) (string, error) {
	if err := ensureMinioClient(); err != nil {
		return "", fmt.Errorf("failed to initialize MinIO client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	url, err := minioClient.PresignedGetObject(ctx, bucket, objectName, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return url.String(), nil
}
