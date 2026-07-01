package service

import (
	"errors"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"aftersalescore/internal/dto"
	"aftersalescore/internal/model"
	"aftersalescore/internal/repo"
	"aftersalescore/internal/storage"

	"gorm.io/gorm"
)

type UnboxingService struct {
	repos    *repo.Repos
	store    storage.Storage
	tenantID uint64
}

func NewUnboxingService(repos *repo.Repos, store storage.Storage) *UnboxingService {
	return &UnboxingService{repos: repos, store: store}
}

func (s *UnboxingService) ForTenant(tenantID uint64) *UnboxingService {
	return &UnboxingService{repos: s.repos, store: s.store, tenantID: repo.NormalizeTenantID(tenantID)}
}

func (s *UnboxingService) List(f repo.UnboxingListFilter) ([]dto.UnboxingListItem, int64, error) {
	list, total, err := s.repos.Unboxing.ForTenant(s.tenantID).List(f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]dto.UnboxingListItem, 0, len(list))
	ur := s.repos.Unboxing.ForTenant(s.tenantID)
	for _, rec := range list {
		item := dto.UnboxingListItem{
			ID: rec.ID, TrackingNo: rec.TrackingNo, Status: rec.Status,
			VideoURL: rec.VideoURL, VideoDurationSec: rec.VideoDurationSec,
			OperatorName: rec.OperatorName, CreatedAt: formatTime(rec.CreatedAt),
		}
		if n, err := ur.CountPhotos(rec.ID); err == nil {
			item.PhotoCount = int(n)
		}
		out = append(out, item)
	}
	return out, total, nil
}

func (s *UnboxingService) Get(id uint64) (*dto.UnboxingDetail, error) {
	rec, err := s.repos.Unboxing.ForTenant(s.tenantID).GetWithPhotos(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.toDetail(rec), nil
}

func (s *UnboxingService) Create(in *dto.UnboxingCreateInput, operatorID uint64, operatorName string) (*dto.UnboxingDetail, error) {
	trackingNo := repo.NormalizeTrackingNo(in.TrackingNo)
	if trackingNo == "" {
		return nil, ErrBadRequest
	}
	rec := &model.UnboxingRecord{
		TrackingNo: trackingNo, Status: model.UnboxingStatusDraft,
		Remark: in.Remark, OperatorID: operatorID, OperatorName: operatorName,
	}
	if err := s.repos.Unboxing.ForTenant(s.tenantID).Create(rec); err != nil {
		return nil, err
	}
	return s.Get(rec.ID)
}

func (s *UnboxingService) UploadVideo(id uint64, file *multipart.FileHeader, durationSec int) (*dto.UnboxingDetail, error) {
	ur := s.repos.Unboxing.ForTenant(s.tenantID)
	rec, err := ur.GetWithPhotos(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if rec.Status == model.UnboxingStatusCompleted {
		return nil, ErrInvalidStatus
	}
	subdir := fmt.Sprintf("videos/%d", rec.ID)
	objectKey, url, err := s.store.Upload(file, subdir)
	if err != nil {
		return nil, err
	}
	rec.VideoObjectKey = objectKey
	rec.VideoURL = url
	rec.VideoSize = file.Size
	rec.VideoMimeType = file.Header.Get("Content-Type")
	if rec.VideoMimeType == "" {
		ext := strings.ToLower(filepath.Ext(file.Filename))
		switch ext {
		case ".webm":
			rec.VideoMimeType = "video/webm"
		case ".mp4":
			rec.VideoMimeType = "video/mp4"
		default:
			rec.VideoMimeType = "video/webm"
		}
	}
	if durationSec > 0 {
		rec.VideoDurationSec = durationSec
	}
	if err := ur.Save(rec); err != nil {
		return nil, err
	}
	return s.Get(id)
}

func (s *UnboxingService) UploadPhoto(id uint64, file *multipart.FileHeader, issueRemark string) (*dto.UnboxingDetail, error) {
	ur := s.repos.Unboxing.ForTenant(s.tenantID)
	rec, err := ur.GetWithPhotos(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	sortOrder, err := ur.NextPhotoSort(rec.ID)
	if err != nil {
		return nil, err
	}
	subdir := fmt.Sprintf("photos/%d", rec.ID)
	objectKey, url, err := s.store.Upload(file, subdir)
	if err != nil {
		return nil, err
	}
	photo := &model.UnboxingPhoto{
		RecordID: rec.ID, ObjectKey: objectKey, PhotoURL: url,
		IssueRemark: strings.TrimSpace(issueRemark), SortOrder: sortOrder,
	}
	if err := ur.AddPhoto(photo); err != nil {
		return nil, err
	}
	return s.Get(id)
}

func (s *UnboxingService) Complete(id uint64, in *dto.UnboxingCompleteInput) (*dto.UnboxingDetail, error) {
	ur := s.repos.Unboxing.ForTenant(s.tenantID)
	rec, err := ur.GetWithPhotos(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if rec.VideoURL == "" {
		return nil, fmt.Errorf("请先上传开箱视频")
	}
	if rec.Status == model.UnboxingStatusCompleted {
		return s.toDetail(rec), nil
	}
	if in != nil {
		if in.VideoDurationSec > 0 {
			rec.VideoDurationSec = in.VideoDurationSec
		}
		if in.Remark != "" {
			rec.Remark = in.Remark
		}
	}
	rec.Status = model.UnboxingStatusCompleted
	if err := ur.Save(rec); err != nil {
		return nil, err
	}
	return s.Get(id)
}

func (s *UnboxingService) VideoDownload(id uint64) (*dto.UnboxingDownloadResp, error) {
	rec, err := s.repos.Unboxing.ForTenant(s.tenantID).GetWithPhotos(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if rec.VideoURL == "" {
		return nil, ErrBadRequest
	}
	ext := ".webm"
	if strings.Contains(rec.VideoMimeType, "mp4") {
		ext = ".mp4"
	}
	return &dto.UnboxingDownloadResp{
		URL:      rec.VideoURL,
		Filename: fmt.Sprintf("unboxing_%s_%d%s", rec.TrackingNo, rec.ID, ext),
	}, nil
}

func (s *UnboxingService) toDetail(rec *model.UnboxingRecord) *dto.UnboxingDetail {
	detail := &dto.UnboxingDetail{
		ID: rec.ID, TrackingNo: rec.TrackingNo, VideoURL: rec.VideoURL,
		VideoSize: rec.VideoSize, VideoDurationSec: rec.VideoDurationSec,
		VideoMimeType: rec.VideoMimeType, Status: rec.Status, Remark: rec.Remark,
		OperatorID: rec.OperatorID, OperatorName: rec.OperatorName,
		CreatedAt: formatTime(rec.CreatedAt),
		Photos: make([]dto.UnboxingPhotoDetail, 0, len(rec.Photos)),
	}
	for _, p := range rec.Photos {
		detail.Photos = append(detail.Photos, dto.UnboxingPhotoDetail{
			ID: p.ID, PhotoURL: p.PhotoURL, IssueRemark: p.IssueRemark,
			SortOrder: p.SortOrder, CreatedAt: formatTime(p.CreatedAt),
		})
	}
	return detail
}

func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}
