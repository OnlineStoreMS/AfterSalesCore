package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"aftersalescore/internal/config"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOStorage struct {
	client     *minio.Client
	bucket     string
	baseURL    string
	rootPrefix string
}

func NewMinIO(cfg *config.StorageConfig) (*MinIOStorage, error) {
	m := cfg.MinIO
	if m.Endpoint == "" {
		return nil, fmt.Errorf("minio endpoint required")
	}
	client, err := minio.New(m.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(m.AccessKey, m.SecretKey, ""),
		Secure: m.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client: %w", err)
	}
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, m.Bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err := client.MakeBucket(ctx, m.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, err
		}
	}
	if m.PublicRead {
		_ = setBucketPublicRead(ctx, client, m.Bucket)
	}
	baseURL := strings.TrimRight(cfg.PublicBaseURL, "/")
	if baseURL == "" {
		scheme := "http"
		if m.UseSSL {
			scheme = "https"
		}
		baseURL = fmt.Sprintf("%s://%s/%s", scheme, m.Endpoint, m.Bucket)
	}
	prefix := strings.Trim(m.Prefix, "/")
	if prefix == "" {
		prefix = "unboxing"
	}
	return &MinIOStorage{client: client, bucket: m.Bucket, baseURL: baseURL, rootPrefix: prefix}, nil
}

func setBucketPublicRead(ctx context.Context, client *minio.Client, bucket string) error {
	policy := map[string]any{
		"Version": "2012-10-17",
		"Statement": []map[string]any{{
			"Effect":    "Allow",
			"Principal": map[string]any{"AWS": []string{"*"}},
			"Action":    []string{"s3:GetObject"},
			"Resource":  []string{fmt.Sprintf("arn:aws:s3:::%s/*", bucket)},
		}},
	}
	raw, err := json.Marshal(policy)
	if err != nil {
		return err
	}
	return client.SetBucketPolicy(ctx, bucket, string(raw))
}

func (s *MinIOStorage) objectKey(subdir, filename string) string {
	subdir = strings.Trim(subdir, "/")
	key := s.rootPrefix
	if subdir != "" {
		key += "/" + subdir
	}
	return key + "/" + safeFilename(filename)
}

func (s *MinIOStorage) Upload(file *multipart.FileHeader, subdir string) (string, string, error) {
	src, err := file.Open()
	if err != nil {
		return "", "", err
	}
	defer src.Close()
	ext := filepath.Ext(file.Filename)
	name := fmt.Sprintf("%s_%s%s", time.Now().Format("20060102150405"), uuid.New().String()[:8], ext)
	contentType := file.Header.Get("Content-Type")
	return s.UploadReader(src, file.Size, subdir, name, contentType)
}

func (s *MinIOStorage) UploadReader(r io.Reader, size int64, subdir, filename, contentType string) (string, string, error) {
	key := s.objectKey(subdir, filename)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	_, err := s.client.PutObject(context.Background(), s.bucket, key, r, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", "", err
	}
	return key, s.PublicURL(key), nil
}

func (s *MinIOStorage) PublicURL(objectKey string) string {
	objectKey = strings.TrimPrefix(objectKey, "/")
	return s.baseURL + "/" + objectKey
}

func (s *MinIOStorage) GetObject(objectKey string) (*minio.Object, error) {
	objectKey = strings.TrimPrefix(strings.TrimSpace(objectKey), "/")
	return s.client.GetObject(context.Background(), s.bucket, objectKey, minio.GetObjectOptions{})
}
