package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"aftersalescore/internal/config"
	"aftersalescore/internal/dto"
	"aftersalescore/internal/integrations/storesyncagent"
	"aftersalescore/internal/model"
	"aftersalescore/internal/repo"
	"aftersalescore/internal/storage"

	"gorm.io/gorm"
)

type EdgeRecordService struct {
	repos       *repo.Repos
	store       storage.Storage
	media       *storage.EdgeMediaResolver
	objects     *storage.EdgeObjectStore
	storeSync   *storesyncagent.Client
	cloudEdgeID string
}

type EdgeRecordVideoFile struct {
	Reader      io.ReadCloser
	ContentType string
	Filename    string
}

func NewEdgeRecordService(repos *repo.Repos, store storage.Storage, media *storage.EdgeMediaResolver, objects *storage.EdgeObjectStore, storeSync *storesyncagent.Client, cfg *config.Config) *EdgeRecordService {
	return &EdgeRecordService{
		repos: repos, store: store, media: media, objects: objects, storeSync: storeSync,
		cloudEdgeID: cfg.Edge.CloudEdgeID,
	}
}

func (s *EdgeRecordService) List(ctx context.Context, bearerToken string, f repo.EdgeRecordListFilter) ([]dto.EdgeRecordListItem, int64, error) {
	list, total, err := s.repos.EdgeRecord.List(f)
	if err != nil {
		return nil, 0, err
	}
	nameMap := s.edgeNameMap()
	out := make([]dto.EdgeRecordListItem, 0, len(list))
	for _, rec := range list {
		out = append(out, s.toListItem(&rec, nameMap))
	}
	if f.WithGoods {
		s.attachGoods(ctx, bearerToken, f.Type, out)
	}
	return out, total, nil
}

func (s *EdgeRecordService) Get(id int64) (*dto.EdgeRecordDetail, error) {
	rec, err := s.repos.EdgeRecord.Get(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	nameMap := s.edgeNameMap()
	return s.toDetail(rec, nameMap), nil
}

func (s *EdgeRecordService) Create(in *dto.EdgeRecordCreateInput) (*dto.EdgeRecordDetail, error) {
	recordType := strings.TrimSpace(in.Type)
	if recordType != model.EdgeRecordTypePacking && recordType != model.EdgeRecordTypeUnboxing {
		return nil, ErrBadRequest
	}
	trackingNo := repo.NormalizeTrackingNo(in.TrackingNo)
	if trackingNo == "" {
		return nil, ErrBadRequest
	}
	rec := &model.EdgeRecord{
		EdgeID: s.cloudEdgeID, Type: recordType,
		TrackingNumber: trackingNo, Status: model.EdgeRecordStatusDraft,
		Remark: strings.TrimSpace(in.Remark), PhotoPaths: model.StringSlice{},
	}
	if err := s.repos.EdgeRecord.Create(rec); err != nil {
		return nil, err
	}
	return s.Get(rec.ID)
}

func (s *EdgeRecordService) UploadVideo(id int64, file *multipart.FileHeader, durationSec int) (*dto.EdgeRecordDetail, error) {
	rec, err := s.getEditableCloudRecord(id)
	if err != nil {
		return nil, err
	}
	subdir := fmt.Sprintf("cloud/%s/videos/%d", rec.Type, rec.ID)
	objectKey, _, err := s.store.Upload(file, subdir)
	if err != nil {
		return nil, err
	}
	rec.VideoPath = objectKey
	if err := s.repos.EdgeRecord.Save(rec); err != nil {
		return nil, err
	}
	_ = durationSec
	return s.Get(id)
}

func (s *EdgeRecordService) UploadPhoto(id int64, file *multipart.FileHeader) (*dto.EdgeRecordDetail, error) {
	rec, err := s.getEditableCloudRecord(id)
	if err != nil {
		return nil, err
	}
	subdir := fmt.Sprintf("cloud/%s/photos/%d", rec.Type, rec.ID)
	objectKey, _, err := s.store.Upload(file, subdir)
	if err != nil {
		return nil, err
	}
	rec.PhotoPaths = append(rec.PhotoPaths, objectKey)
	if err := s.repos.EdgeRecord.Save(rec); err != nil {
		return nil, err
	}
	return s.Get(id)
}

func (s *EdgeRecordService) Complete(id int64, in *dto.EdgeRecordCompleteInput) (*dto.EdgeRecordDetail, error) {
	rec, err := s.getEditableCloudRecord(id)
	if err != nil {
		return nil, err
	}
	if rec.VideoPath == "" {
		return nil, fmt.Errorf("请先完成视频录制")
	}
	if rec.Status == model.EdgeRecordStatusCompleted {
		return s.Get(id)
	}
	if in != nil && strings.TrimSpace(in.Remark) != "" {
		rec.Remark = strings.TrimSpace(in.Remark)
	}
	now := time.Now()
	rec.Status = model.EdgeRecordStatusCompleted
	rec.CompletedAt = &now
	if err := s.repos.EdgeRecord.Save(rec); err != nil {
		return nil, err
	}
	return s.Get(id)
}

func (s *EdgeRecordService) Delete(id int64) error {
	_, err := s.repos.EdgeRecord.Get(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	return s.repos.EdgeRecord.Delete(id)
}

func (s *EdgeRecordService) BatchDelete(ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, ErrBadRequest
	}
	deleted := 0
	for _, id := range ids {
		if err := s.Delete(id); err == nil {
			deleted++
		}
	}
	return deleted, nil
}

func (s *EdgeRecordService) OpenVideoDownload(id int64) (*EdgeRecordVideoFile, error) {
	rec, err := s.repos.EdgeRecord.Get(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if rec.VideoPath == "" {
		return nil, ErrBadRequest
	}
	if s.objects == nil {
		return nil, fmt.Errorf("video download unavailable")
	}
	reader, contentType, err := s.objects.Open(rec.EdgeID, rec.VideoPath)
	if err != nil {
		return nil, err
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return &EdgeRecordVideoFile{
		Reader:      reader,
		ContentType: contentType,
		Filename:    s.videoDownloadFilename(rec),
	}, nil
}

func (s *EdgeRecordService) videoDownloadFilename(rec *model.EdgeRecord) string {
	ext := filepath.Ext(rec.VideoPath)
	if ext == "" {
		ext = ".webm"
	}
	typeLabel := "打包"
	if rec.Type == model.EdgeRecordTypeUnboxing {
		typeLabel = "开箱"
	}
	return fmt.Sprintf("%s_%s_%d%s", typeLabel, rec.TrackingNumber, rec.ID, ext)
}

func (s *EdgeRecordService) DashboardStats() (map[string]int64, error) {
	unboxing, err := s.repos.EdgeRecord.CountByType(model.EdgeRecordTypeUnboxing)
	if err != nil {
		return nil, err
	}
	packing, err := s.repos.EdgeRecord.CountByType(model.EdgeRecordTypePacking)
	if err != nil {
		return nil, err
	}
	return map[string]int64{
		"unboxingCount": unboxing,
		"packingCount":  packing,
	}, nil
}

func (s *EdgeRecordService) getEditableCloudRecord(id int64) (*model.EdgeRecord, error) {
	rec, err := s.repos.EdgeRecord.Get(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if rec.EdgeID != s.cloudEdgeID {
		return nil, ErrInvalidStatus
	}
	if rec.Status == model.EdgeRecordStatusCompleted {
		return nil, ErrInvalidStatus
	}
	return rec, nil
}

func (s *EdgeRecordService) edgeNameMap() map[string]string {
	m := map[string]string{s.cloudEdgeID: "云端浏览器"}
	devs, _ := s.repos.EdgeDevice.List()
	for _, d := range devs {
		m[d.EdgeID] = d.Name
	}
	return m
}

func (s *EdgeRecordService) toListItem(rec *model.EdgeRecord, nameMap map[string]string) dto.EdgeRecordListItem {
	item := dto.EdgeRecordListItem{
		ID: rec.ID, EdgeID: rec.EdgeID, EdgeName: nameMap[rec.EdgeID],
		Type: rec.Type, TrackingNo: rec.TrackingNumber, Status: rec.Status,
		VideoURL: s.media.MediaURL(rec.EdgeID, rec.VideoPath),
		PhotoCount: len(rec.PhotoPaths), Remark: rec.Remark,
		CreatedAt: formatTime(rec.CreatedAt),
	}
	if rec.CompletedAt != nil {
		item.CompletedAt = formatTime(*rec.CompletedAt)
	}
	return item
}

func (s *EdgeRecordService) toDetail(rec *model.EdgeRecord, nameMap map[string]string) *dto.EdgeRecordDetail {
	detail := &dto.EdgeRecordDetail{
		ID: rec.ID, EdgeID: rec.EdgeID, EdgeName: nameMap[rec.EdgeID],
		Type: rec.Type, TrackingNo: rec.TrackingNumber, Status: rec.Status,
		VideoURL: s.media.MediaURL(rec.EdgeID, rec.VideoPath),
		Remark: rec.Remark, CreatedAt: formatTime(rec.CreatedAt),
		UpdatedAt: formatTime(rec.UpdatedAt),
		Photos: make([]dto.EdgeRecordPhotoDetail, 0, len(rec.PhotoPaths)),
	}
	for _, p := range rec.PhotoPaths {
		detail.Photos = append(detail.Photos, dto.EdgeRecordPhotoDetail{
			URL: s.media.MediaURL(rec.EdgeID, p),
		})
	}
	if rec.CompletedAt != nil {
		detail.CompletedAt = formatTime(*rec.CompletedAt)
	}
	return detail
}

func (s *EdgeRecordService) attachGoods(ctx context.Context, bearerToken, recordType string, items []dto.EdgeRecordListItem) {
	if s.storeSync == nil || !s.storeSync.Enabled() || len(items) == 0 {
		return
	}
	trackingNos := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		key := repo.NormalizeTrackingNo(item.TrackingNo)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		trackingNos = append(trackingNos, key)
	}
	if len(trackingNos) == 0 {
		return
	}
	lookups, err := s.storeSync.LookupByTrackingNos(ctx, bearerToken, recordType, trackingNos)
	if err != nil {
		return
	}
	for i := range items {
		key := repo.NormalizeTrackingNo(items[i].TrackingNo)
		lookup, ok := lookups[key]
		if !ok || !lookup.Found || len(lookup.Goods) == 0 {
			continue
		}
		items[i].Goods = toEdgeRecordGoods(lookup.Goods)
	}
}

func toEdgeRecordGoods(goods []storesyncagent.TradeGoods) []dto.EdgeRecordGoods {
	out := make([]dto.EdgeRecordGoods, 0, len(goods))
	for _, g := range goods {
		out = append(out, dto.EdgeRecordGoods{
			Title:   g.Title,
			SkuName: g.SkuName,
			PicURL:  g.PicURL,
			Num:     g.Num,
		})
	}
	return out
}
