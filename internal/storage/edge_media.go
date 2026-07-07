package storage

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"aftersalescore/internal/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// EdgeMediaResolver builds accessible URLs (presigned) for edge_record media paths.
type EdgeMediaResolver struct {
	client       *minio.Client
	cloudBucket  string
	edgeBucket   string
	cloudBaseURL string
	edgeBaseURL  string
	cloudEdgeID  string
	presignTTL   time.Duration
}

func NewEdgeMediaResolver(cfg *config.Config, cloudStore Storage) (*EdgeMediaResolver, error) {
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

	client, err := minio.New(endpoint, &minio.Options{
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
	cloudBucket := sm.Bucket
	if cloudBucket == "" {
		cloudBucket = "aftersalescore"
	}

	ctx := context.Background()
	if em.PublicRead {
		_ = setBucketPublicRead(ctx, client, edgeBucket)
	}

	cloudBase := strings.TrimRight(cfg.Storage.PublicBaseURL, "/")
	if ms, ok := cloudStore.(*MinIOStorage); ok {
		cloudBase = strings.TrimRight(ms.baseURL, "/")
	}
	edgeBase := strings.TrimRight(em.PublicBaseURL, "/")
	if edgeBase == "" {
		scheme := "http"
		if useSSL {
			scheme = "https"
		}
		edgeBase = fmt.Sprintf("%s://%s/%s", scheme, endpoint, edgeBucket)
	}

	return &EdgeMediaResolver{
		client:       client,
		cloudBucket:  cloudBucket,
		edgeBucket:   edgeBucket,
		cloudBaseURL: cloudBase,
		edgeBaseURL:  edgeBase,
		cloudEdgeID:  cfg.Edge.CloudEdgeID,
		presignTTL:   24 * time.Hour,
	}, nil
}

func (r *EdgeMediaResolver) IsCloudEdge(edgeID string) bool {
	return edgeID == r.cloudEdgeID || edgeID == ""
}

func (r *EdgeMediaResolver) MediaURL(edgeID, relPath string) string {
	relPath = strings.TrimPrefix(strings.TrimSpace(relPath), "/")
	if relPath == "" {
		return ""
	}
	bucket := r.edgeBucket
	fallbackBase := r.edgeBaseURL
	if r.IsCloudEdge(edgeID) || strings.HasPrefix(relPath, "cloud/") {
		bucket = r.cloudBucket
		fallbackBase = r.cloudBaseURL
	}

	u, err := r.client.PresignedGetObject(context.Background(), bucket, relPath, r.presignTTL, url.Values{})
	if err != nil {
		return fallbackBase + "/" + relPath
	}
	return u.String()
}
