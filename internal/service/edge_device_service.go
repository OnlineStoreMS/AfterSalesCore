package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"aftersalescore/internal/config"
	"aftersalescore/internal/dto"
	"aftersalescore/internal/model"
	"aftersalescore/internal/repo"

	"gorm.io/gorm"
)

type EdgeDeviceService struct {
	repos  *repo.Repos
	cfg    *config.Config
	client *http.Client
}

func NewEdgeDeviceService(repos *repo.Repos, cfg *config.Config) *EdgeDeviceService {
	return &EdgeDeviceService{
		repos: repos, cfg: cfg,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

func (s *EdgeDeviceService) List() ([]dto.EdgeDeviceItem, error) {
	list, err := s.repos.EdgeDevice.List()
	if err != nil {
		return nil, err
	}
	out := make([]dto.EdgeDeviceItem, 0, len(list))
	for _, d := range list {
		out = append(out, s.toItem(&d))
	}
	return out, nil
}

func (s *EdgeDeviceService) Create(in *dto.EdgeDeviceCreateInput) (*dto.EdgeDeviceItem, error) {
	edgeID := strings.TrimSpace(in.EdgeID)
	if edgeID == "" || edgeID == s.cfg.Edge.CloudEdgeID {
		return nil, ErrBadRequest
	}
	if _, err := s.repos.EdgeDevice.GetByEdgeID(edgeID); err == nil {
		return nil, fmt.Errorf("edge_id 已存在")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	dev := &model.EdgeDevice{
		EdgeID: edgeID, Name: strings.TrimSpace(in.Name),
		BaseURL: strings.TrimRight(strings.TrimSpace(in.BaseURL), "/"),
		Status: model.EdgeDeviceStatusUnknown, Remark: strings.TrimSpace(in.Remark),
	}
	if dev.Name == "" {
		return nil, ErrBadRequest
	}
	if err := s.repos.EdgeDevice.Create(dev); err != nil {
		return nil, err
	}
	if dev.BaseURL != "" {
		s.probeDevice(dev)
	}
	item := s.toItem(dev)
	return &item, nil
}

func (s *EdgeDeviceService) Update(id uint64, in *dto.EdgeDeviceUpdateInput) (*dto.EdgeDeviceItem, error) {
	dev, err := s.repos.EdgeDevice.Get(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if in.Name != "" {
		dev.Name = strings.TrimSpace(in.Name)
	}
	if in.BaseURL != "" {
		dev.BaseURL = strings.TrimRight(strings.TrimSpace(in.BaseURL), "/")
	}
	if in.Remark != "" {
		dev.Remark = strings.TrimSpace(in.Remark)
	}
	if err := s.repos.EdgeDevice.Save(dev); err != nil {
		return nil, err
	}
	item := s.toItem(dev)
	return &item, nil
}

func (s *EdgeDeviceService) Delete(id uint64) error {
	dev, err := s.repos.EdgeDevice.Get(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if dev.EdgeID == s.cfg.Edge.CloudEdgeID {
		return ErrInvalidStatus
	}
	return s.repos.EdgeDevice.Delete(id)
}

func (s *EdgeDeviceService) Probe(id uint64) (*dto.EdgeDeviceItem, error) {
	dev, err := s.repos.EdgeDevice.Get(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if dev.BaseURL == "" {
		return nil, fmt.Errorf("未配置 base_url，无法探测在线状态")
	}
	s.probeDevice(dev)
	item := s.toItem(dev)
	return &item, nil
}

func (s *EdgeDeviceService) SyncFromRecords() error {
	ids, err := s.repos.EdgeRecord.DistinctEdgeIDs()
	if err != nil {
		return err
	}
	for _, edgeID := range ids {
		if edgeID == "" || edgeID == s.cfg.Edge.CloudEdgeID {
			continue
		}
		_ = s.repos.EdgeDevice.UpsertByEdgeID(edgeID, edgeID)
	}
	return nil
}

func (s *EdgeDeviceService) StartHealthPoller(ctx context.Context) {
	interval := time.Duration(s.cfg.Edge.HealthPollSec) * time.Second
	if interval < 10*time.Second {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	s.pollAll()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.SyncFromRecords()
			s.pollAll()
		}
	}
}

func (s *EdgeDeviceService) pollAll() {
	list, err := s.repos.EdgeDevice.ListProbeable()
	if err != nil {
		return
	}
	for i := range list {
		s.probeDevice(&list[i])
	}
}

func (s *EdgeDeviceService) probeDevice(dev *model.EdgeDevice) {
	online := false
	now := time.Now()

	healthURL := dev.BaseURL + "/api/health"
	req, err := http.NewRequest(http.MethodGet, healthURL, nil)
	if err == nil {
		resp, err := s.client.Do(req)
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK && strings.Contains(string(body), "ok") {
				online = true
			}
		}
	}

	if online {
		infoURL := dev.BaseURL + "/api/info"
		if req, err := http.NewRequest(http.MethodGet, infoURL, nil); err == nil {
			if resp, err := s.client.Do(req); err == nil {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				var info struct {
					EdgeID   string `json:"edge_id"`
					EdgeName string `json:"edge_name"`
				}
				if json.Unmarshal(body, &info) == nil {
					if info.EdgeName != "" {
						dev.Name = info.EdgeName
					}
					if info.EdgeID != "" && info.EdgeID != dev.EdgeID {
						dev.EdgeID = info.EdgeID
					}
				}
			}
		}
		dev.Status = model.EdgeDeviceStatusOnline
		dev.LastSeenAt = &now
	} else {
		dev.Status = model.EdgeDeviceStatusOffline
	}
	_ = s.repos.EdgeDevice.Save(dev)
}

func (s *EdgeDeviceService) toItem(dev *model.EdgeDevice) dto.EdgeDeviceItem {
	item := dto.EdgeDeviceItem{
		ID: dev.ID, EdgeID: dev.EdgeID, Name: dev.Name,
		BaseURL: dev.BaseURL, Status: dev.Status, Remark: dev.Remark,
		CreatedAt: formatTime(dev.CreatedAt), UpdatedAt: formatTime(dev.UpdatedAt),
	}
	if dev.LastSeenAt != nil {
		item.LastSeenAt = formatTime(*dev.LastSeenAt)
	}
	return item
}

func (s *EdgeDeviceService) EnsureDefaults() error {
	cloudID := s.cfg.Edge.CloudEdgeID
	if _, err := s.repos.EdgeDevice.GetByEdgeID(cloudID); errors.Is(err, gorm.ErrRecordNotFound) {
		return s.repos.EdgeDevice.Create(&model.EdgeDevice{
			EdgeID: cloudID, Name: "云端浏览器",
			Status: model.EdgeDeviceStatusOnline,
			Remark: "云端 AfterSalesCore 浏览器录制",
		})
	}
	return nil
}
