package storage

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aftersalescore/internal/config"

	"github.com/google/uuid"
)

type Storage interface {
	Upload(file *multipart.FileHeader, subdir string) (objectKey, publicURL string, err error)
	UploadReader(r io.Reader, size int64, subdir, filename, contentType string) (objectKey, publicURL string, err error)
	PublicURL(objectKey string) string
}

type LocalStorage struct {
	baseDir string
	baseURL string
	prefix  string
}

func NewLocal(cfg *config.StorageConfig) (*LocalStorage, error) {
	base := filepath.Join(cfg.LocalPath, cfg.Prefix)
	if err := os.MkdirAll(base, 0o755); err != nil {
		return nil, err
	}
	return &LocalStorage{
		baseDir: base,
		baseURL: strings.TrimRight(cfg.PublicBaseURL, "/"),
		prefix:  cfg.Prefix,
	}, nil
}

func (s *LocalStorage) Upload(file *multipart.FileHeader, subdir string) (string, string, error) {
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

func (s *LocalStorage) UploadReader(r io.Reader, size int64, subdir, filename, _ string) (string, string, error) {
	subdir = strings.Trim(subdir, "/")
	rel := filepath.Join(subdir, safeFilename(filename))
	destPath := filepath.Join(s.baseDir, rel)
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return "", "", err
	}
	dst, err := os.Create(destPath)
	if err != nil {
		return "", "", err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, r); err != nil {
		return "", "", err
	}
	objectKey := strings.ReplaceAll(filepath.Join(s.prefix, rel), "\\", "/")
	return objectKey, s.baseURL + "/" + objectKey, nil
}

func (s *LocalStorage) PublicURL(objectKey string) string {
	objectKey = strings.TrimPrefix(objectKey, "/")
	return s.baseURL + "/" + objectKey
}

func New(cfg *config.StorageConfig) (Storage, error) {
	switch cfg.Driver {
	case "minio":
		return NewMinIO(cfg)
	case "local", "":
		return NewLocal(cfg)
	default:
		return nil, fmt.Errorf("unsupported storage driver: %s", cfg.Driver)
	}
}

func safeFilename(name string) string {
	name = filepath.Base(name)
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, name)
	if name == "" || name == "." {
		return "file.bin"
	}
	return name
}
