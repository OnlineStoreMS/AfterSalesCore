package storage

import (
	"strings"

	"aftersalescore/internal/config"
)

// EdgeMediaResolver builds public URLs for edge_record media paths.
type EdgeMediaResolver struct {
	cloudBaseURL string
	edgeBaseURL  string
	cloudEdgeID  string
}

func NewEdgeMediaResolver(cfg *config.Config, cloudStore Storage) *EdgeMediaResolver {
	cloudBase := strings.TrimRight(cfg.Storage.PublicBaseURL, "/")
	if ms, ok := cloudStore.(*MinIOStorage); ok {
		cloudBase = strings.TrimRight(ms.baseURL, "/")
	}
	edgeBase := strings.TrimRight(cfg.Edge.MinIO.PublicBaseURL, "/")
	return &EdgeMediaResolver{
		cloudBaseURL: cloudBase,
		edgeBaseURL:  edgeBase,
		cloudEdgeID:  cfg.Edge.CloudEdgeID,
	}
}

func (r *EdgeMediaResolver) IsCloudEdge(edgeID string) bool {
	return edgeID == r.cloudEdgeID || edgeID == ""
}

func (r *EdgeMediaResolver) MediaURL(edgeID, relPath string) string {
	relPath = strings.TrimPrefix(strings.TrimSpace(relPath), "/")
	if relPath == "" {
		return ""
	}
	if r.IsCloudEdge(edgeID) || strings.HasPrefix(relPath, "cloud/") {
		return r.cloudBaseURL + "/" + relPath
	}
	return r.edgeBaseURL + "/" + relPath
}
