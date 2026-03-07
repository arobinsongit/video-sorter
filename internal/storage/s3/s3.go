package s3

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

	"video-sorter/internal/storage"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Storage implements storage.Provider for Amazon S3.
type Storage struct {
	client *s3.Client
	region string
}

// Credentials holds access key and region for S3.
type Credentials struct {
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
	Region          string `json:"region"`
	Endpoint        string `json:"endpoint,omitempty"`
}

// CredsPath returns the path to the stored S3 credentials.
func CredsPath() string {
	return filepath.Join(storage.CredentialsDir(), "s3-credentials.json")
}

// LoadCreds reads saved S3 credentials from disk.
func LoadCreds() (*Credentials, error) {
	data, err := os.ReadFile(CredsPath())
	if err != nil {
		return nil, err
	}
	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}

// New creates an S3 storage provider from saved credentials.
func New() (*Storage, error) {
	creds, err := LoadCreds()
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
	return &Storage{client: client, region: creds.Region}, nil
}

func parseS3Path(fullPath string) (bucket, key string) {
	path := strings.TrimPrefix(fullPath, "s3://")
	parts := strings.SplitN(path, "/", 2)
	bucket = parts[0]
	if len(parts) > 1 {
		key = parts[1]
	}
	return
}

func (st *Storage) ListFiles(path string) ([]storage.FileInfo, error) {
	bucket, prefix := parseS3Path(path)
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	}

	var files []storage.FileInfo
	paginator := s3.NewListObjectsV2Paginator(st.client, input)
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
			if !storage.MediaExts[ext] {
				continue
			}
			modified := ""
			if obj.LastModified != nil {
				modified = obj.LastModified.Format("2006-01-02 15:04")
			}
			files = append(files, storage.FileInfo{
				Name:     name,
				Size:     *obj.Size,
				Modified: modified,
			})
		}
	}
	return files, nil
}

func (st *Storage) ServeFile(w http.ResponseWriter, r *http.Request, dir, file string) {
	bucket, prefix := parseS3Path(dir)
	key := prefix
	if key != "" && !strings.HasSuffix(key, "/") {
		key += "/"
	}
	key += file

	result, err := st.client.GetObject(context.Background(), &s3.GetObjectInput{
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

func (st *Storage) ReadFile(path string) ([]byte, error) {
	bucket, key := parseS3Path(path)
	result, err := st.client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer result.Body.Close()
	return io.ReadAll(result.Body)
}

func (st *Storage) WriteFile(path string, data []byte) error {
	bucket, key := parseS3Path(path)
	_, err := st.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	return err
}

func (st *Storage) Rename(dir, oldName, newName string) error {
	bucket, prefix := parseS3Path(dir)
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	oldKey := prefix + oldName
	newKey := prefix + newName

	_, err := st.client.CopyObject(context.Background(), &s3.CopyObjectInput{
		Bucket:     aws.String(bucket),
		CopySource: aws.String(bucket + "/" + oldKey),
		Key:        aws.String(newKey),
	})
	if err != nil {
		return err
	}
	_, err = st.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(oldKey),
	})
	return err
}

func (st *Storage) MoveFile(oldPath, newPath string) error {
	bucket, oldKey := parseS3Path(oldPath)
	_, newKey := parseS3Path(newPath)

	_, err := st.client.CopyObject(context.Background(), &s3.CopyObjectInput{
		Bucket:     aws.String(bucket),
		CopySource: aws.String(bucket + "/" + oldKey),
		Key:        aws.String(newKey),
	})
	if err != nil {
		return err
	}
	_, err = st.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(oldKey),
	})
	return err
}

func (st *Storage) CopyFile(oldPath, newPath string) error {
	bucket, oldKey := parseS3Path(oldPath)
	_, newKey := parseS3Path(newPath)

	_, err := st.client.CopyObject(context.Background(), &s3.CopyObjectInput{
		Bucket:     aws.String(bucket),
		CopySource: aws.String(bucket + "/" + oldKey),
		Key:        aws.String(newKey),
	})
	return err
}

func (st *Storage) FileExists(path string) bool {
	bucket, key := parseS3Path(path)
	_, err := st.client.HeadObject(context.Background(), &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	return err == nil
}

func (st *Storage) MkdirAll(path string) error {
	return nil
}

func (st *Storage) IsLocal() bool {
	return false
}

// ListBuckets lists all buckets for the browse UI.
func (st *Storage) ListBuckets() ([]string, error) {
	result, err := st.client.ListBuckets(context.Background(), &s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}
	var names []string
	for _, b := range result.Buckets {
		names = append(names, *b.Name)
	}
	return names, nil
}

// ListFolders lists "folders" (common prefixes) under a bucket/prefix.
func (st *Storage) ListFolders(bucket, prefix string) ([]string, error) {
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	result, err := st.client.ListObjectsV2(context.Background(), &s3.ListObjectsV2Input{
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
