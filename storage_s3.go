package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Storage implements StorageProvider for Amazon S3.
type S3Storage struct {
	client *s3.Client
	region string
}

// s3CredsPath returns the path to the stored S3 credentials.
func s3CredsPath() string {
	return filepath.Join(credentialsDir(), "s3-credentials.json")
}

// S3Credentials holds access key and region for S3.
type S3Credentials struct {
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
	Region          string `json:"region"`
	Endpoint        string `json:"endpoint,omitempty"` // For S3-compatible services
}

// loadS3Creds reads saved S3 credentials from disk.
func loadS3Creds() (*S3Credentials, error) {
	data, err := os.ReadFile(s3CredsPath())
	if err != nil {
		return nil, err
	}
	var creds S3Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}

// newS3Storage creates an S3 storage provider from saved credentials.
func newS3Storage() (*S3Storage, error) {
	creds, err := loadS3Creds()
	if err != nil {
		return nil, fmt.Errorf("no saved S3 credentials: %w", err)
	}

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(creds.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			creds.AccessKeyID, creds.SecretAccessKey, "",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load S3 config: %w", err)
	}

	opts := func(o *s3.Options) {}
	if creds.Endpoint != "" {
		opts = func(o *s3.Options) {
			o.BaseEndpoint = aws.String(creds.Endpoint)
			o.UsePathStyle = true
		}
	}

	client := s3.NewFromConfig(cfg, opts)
	return &S3Storage{client: client, region: creds.Region}, nil
}

// parseS3Path splits "s3://bucket/prefix/path" into bucket and key.
func parseS3Path(fullPath string) (bucket, key string) {
	path := strings.TrimPrefix(fullPath, "s3://")
	parts := strings.SplitN(path, "/", 2)
	bucket = parts[0]
	if len(parts) > 1 {
		key = parts[1]
	}
	return
}

func (s *S3Storage) ListFiles(path string) ([]FileInfo, error) {
	bucket, prefix := parseS3Path(path)
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	}

	var files []FileInfo
	paginator := s3.NewListObjectsV2Paginator(s.client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			return nil, err
		}
		for _, obj := range page.Contents {
			name := strings.TrimPrefix(*obj.Key, prefix)
			if name == "" {
				continue
			}
			ext := strings.ToLower(filepath.Ext(name))
			if !mediaExts[ext] {
				continue
			}
			modified := ""
			if obj.LastModified != nil {
				modified = obj.LastModified.Format("2006-01-02 15:04")
			}
			files = append(files, FileInfo{
				Name:     name,
				Size:     *obj.Size,
				Modified: modified,
			})
		}
	}
	return files, nil
}

func (s *S3Storage) ServeFile(w http.ResponseWriter, r *http.Request, dir, file string) {
	bucket, prefix := parseS3Path(dir)
	key := prefix
	if key != "" && !strings.HasSuffix(key, "/") {
		key += "/"
	}
	key += file

	result, err := s.client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		http.Error(w, `{"error":"file not found"}`, http.StatusNotFound)
		return
	}
	defer result.Body.Close()

	if result.ContentType != nil {
		w.Header().Set("Content-Type", *result.ContentType)
	}
	io.Copy(w, result.Body)
}

func (s *S3Storage) ReadFile(path string) ([]byte, error) {
	bucket, key := parseS3Path(path)
	result, err := s.client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer result.Body.Close()
	return io.ReadAll(result.Body)
}

func (s *S3Storage) WriteFile(path string, data []byte) error {
	bucket, key := parseS3Path(path)
	_, err := s.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	return err
}

func (s *S3Storage) Rename(dir, oldName, newName string) error {
	bucket, prefix := parseS3Path(dir)
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	oldKey := prefix + oldName
	newKey := prefix + newName

	// S3 doesn't have rename — copy then delete
	_, err := s.client.CopyObject(context.Background(), &s3.CopyObjectInput{
		Bucket:     aws.String(bucket),
		CopySource: aws.String(bucket + "/" + oldKey),
		Key:        aws.String(newKey),
	})
	if err != nil {
		return err
	}
	_, err = s.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(oldKey),
	})
	return err
}

func (s *S3Storage) MoveFile(oldPath, newPath string) error {
	bucket, oldKey := parseS3Path(oldPath)
	_, newKey := parseS3Path(newPath)

	_, err := s.client.CopyObject(context.Background(), &s3.CopyObjectInput{
		Bucket:     aws.String(bucket),
		CopySource: aws.String(bucket + "/" + oldKey),
		Key:        aws.String(newKey),
	})
	if err != nil {
		return err
	}
	_, err = s.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(oldKey),
	})
	return err
}

func (s *S3Storage) CopyFile(oldPath, newPath string) error {
	bucket, oldKey := parseS3Path(oldPath)
	_, newKey := parseS3Path(newPath)

	_, err := s.client.CopyObject(context.Background(), &s3.CopyObjectInput{
		Bucket:     aws.String(bucket),
		CopySource: aws.String(bucket + "/" + oldKey),
		Key:        aws.String(newKey),
	})
	return err
}

func (s *S3Storage) FileExists(path string) bool {
	bucket, key := parseS3Path(path)
	_, err := s.client.HeadObject(context.Background(), &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	return err == nil
}

func (s *S3Storage) MkdirAll(path string) error {
	// S3 doesn't have directories — they're virtual via key prefixes
	return nil
}

func (s *S3Storage) IsLocal() bool {
	return false
}

// listS3Buckets lists all buckets for the browse UI.
func (s *S3Storage) listBuckets() ([]string, error) {
	result, err := s.client.ListBuckets(context.Background(), &s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}
	var names []string
	for _, b := range result.Buckets {
		names = append(names, *b.Name)
	}
	return names, nil
}

// listFolders lists "folders" (common prefixes) under a bucket/prefix.
func (s *S3Storage) listFolders(bucket, prefix string) ([]string, error) {
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	result, err := s.client.ListObjectsV2(context.Background(), &s3.ListObjectsV2Input{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	})
	if err != nil {
		return nil, err
	}
	var folders []string
	for _, p := range result.CommonPrefixes {
		name := strings.TrimPrefix(*p.Prefix, prefix)
		name = strings.TrimSuffix(name, "/")
		if name != "" {
			folders = append(folders, name)
		}
	}
	return folders, nil
}
