package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"aftersalescore/internal/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// EdgeObjectStore reads edge_record media from cloud or edge MinIO buckets.
type EdgeObjectStore struct {
	cloud       *MinIOStorage
	edgeClient  *minio.Client
	edgeBucket  string
	cloudEdgeID string
}

func NewEdgeObjectStore(cfg *config.Config, cloudStore Storage) (*EdgeObjectStore, error) {
	cloud, ok := cloudStore.(*MinIOStorage)
	if !ok {
		return nil, fmt.Errorf("edge object store requires minio storage")
	}

	em := cfg.Edge.MinIO
	sm := cfg.Storage.MinIO
	endpoint := em.Endpoint
	if endpoint == "" {
		endpoint = sm.Endpoint
	}
	accessKey := em.AccessKey
	if accessKey == "" {
		accessKey = sm.AccessKey
	}
	secretKey := em.SecretKey
	if secretKey == "" {
		secretKey = sm.SecretKey
	}
	useSSL := em.UseSSL
	if endpoint == sm.Endpoint {
		useSSL = sm.UseSSL
	}
	if endpoint == "" {
		return nil, fmt.Errorf("edge minio endpoint required")
	}

	edgeClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("edge minio client: %w", err)
	}

	edgeBucket := em.Bucket
	if edgeBucket == "" {
		edgeBucket = "box-edge"
	}

	return &EdgeObjectStore{
		cloud:       cloud,
		edgeClient:  edgeClient,
		edgeBucket:  edgeBucket,
		cloudEdgeID: cfg.Edge.CloudEdgeID,
	}, nil
}

func (s *EdgeObjectStore) isCloudEdge(edgeID string) bool {
	return edgeID == s.cloudEdgeID || edgeID == ""
}

func (s *EdgeObjectStore) Open(edgeID, relPath string) (io.ReadCloser, string, error) {
	relPath = strings.TrimPrefix(strings.TrimSpace(relPath), "/")
	if relPath == "" {
		return nil, "", fmt.Errorf("empty object path")
	}

	if s.isCloudEdge(edgeID) || strings.HasPrefix(relPath, "cloud/") {
		obj, err := s.cloud.GetObject(relPath)
		if err != nil {
			return nil, "", err
		}
		info, err := obj.Stat()
		if err != nil {
			obj.Close()
			return nil, "", err
		}
		return obj, info.ContentType, nil
	}

	obj, err := s.edgeClient.GetObject(context.Background(), s.edgeBucket, relPath, minio.GetObjectOptions{})
	if err != nil {
		return nil, "", err
	}
	info, err := obj.Stat()
	if err != nil {
		obj.Close()
		return nil, "", err
	}
	return obj, info.ContentType, nil
}
