package repo

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"aftersalescore/internal/dto"
	"aftersalescore/internal/model"

	"gorm.io/gorm"
)

type NotificationRepo struct {
	db *gorm.DB
}

func NewNotificationRepo(db *gorm.DB) *NotificationRepo {
	return &NotificationRepo{db: db}
}

func (r *NotificationRepo) Load(tenantID uint64) (dto.NotificationData, error) {
	row, err := r.getOrCreate(NormalizeTenantID(tenantID))
	if err != nil {
		return dto.NotificationData{}, err
	}
	return rowToNotificationData(*row), nil
}

func (r *NotificationRepo) SaveConfig(tenantID uint64, cfg dto.NotificationConfig) (dto.NotificationData, error) {
	row, err := r.getOrCreate(NormalizeTenantID(tenantID))
	if err != nil {
		return dto.NotificationData{}, err
	}
	data := rowToNotificationData(*row)
	if cfg.Secret == "" {
		cfg.Secret = data.Config.Secret
	}
	if cfg.AppSecret == "" {
		cfg.AppSecret = data.Config.AppSecret
	}
	normalizeNotificationConfig(&cfg)
	if err := r.applyConfig(row, cfg); err != nil {
		return dto.NotificationData{}, err
	}
	if err := r.db.Save(row).Error; err != nil {
		return dto.NotificationData{}, err
	}
	return rowToNotificationData(*row), nil
}

func (r *NotificationRepo) UpdateState(tenantID uint64, fn func(*dto.NotificationState) error) error {
	row, err := r.getOrCreate(NormalizeTenantID(tenantID))
	if err != nil {
		return err
	}
	data := rowToNotificationData(*row)
	if data.State.Notified == nil {
		data.State.Notified = map[string]string{}
	}
	if err := fn(&data.State); err != nil {
		return err
	}
	pruneNotified(data.State.Notified, 60)
	return r.applyState(row, data.State)
}

func (r *NotificationRepo) ResetState(tenantID uint64) (int, error) {
	row, err := r.getOrCreate(NormalizeTenantID(tenantID))
	if err != nil {
		return 0, err
	}
	data := rowToNotificationData(*row)
	cleared := len(data.State.Notified)
	if err := r.applyState(row, dto.NotificationState{Notified: map[string]string{}}); err != nil {
		return 0, err
	}
	return cleared, nil
}

func (r *NotificationRepo) ListTenantIDs() ([]uint64, error) {
	var ids []uint64
	err := r.db.Model(&model.TenantNotification{}).Distinct("tenant_id").Pluck("tenant_id", &ids).Error
	return ids, err
}

func (r *NotificationRepo) getOrCreate(tenantID uint64) (*model.TenantNotification, error) {
	var row model.TenantNotification
	err := r.db.Where("tenant_id = ?", tenantID).First(&row).Error
	if err == nil {
		return &row, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	row = model.TenantNotification{
		TenantID:            tenantID,
		PollIntervalMinutes: 5,
		NotifiedJSON:        "{}",
	}
	if err := r.db.Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *NotificationRepo) applyConfig(row *model.TenantNotification, cfg dto.NotificationConfig) error {
	scenariosJSON, err := json.Marshal(cfg.Scenarios)
	if err != nil {
		return err
	}
	shopIDsJSON, err := json.Marshal(cfg.ShopIDs)
	if err != nil {
		return err
	}
	row.Enabled = cfg.Enabled
	row.WebhookURL = cfg.WebhookURL
	row.Secret = cfg.Secret
	row.PollIntervalMinutes = cfg.PollIntervalMinutes
	row.ScenariosJSON = string(scenariosJSON)
	row.ShopIDsJSON = string(shopIDsJSON)
	row.AppID = cfg.AppID
	row.AppSecret = cfg.AppSecret
	return nil
}

func (r *NotificationRepo) applyState(row *model.TenantNotification, state dto.NotificationState) error {
	if state.Notified == nil {
		state.Notified = map[string]string{}
	}
	notifiedJSON, err := json.Marshal(state.Notified)
	if err != nil {
		return err
	}
	row.LastRunOK = state.LastRunOK
	row.LastError = state.LastError
	row.LastSentCount = state.LastSentCount
	row.LastBarcodeError = state.LastBarcodeError
	row.NotifiedJSON = string(notifiedJSON)
	if state.LastRunAt != "" {
		if t, ok := parseStoreTime(state.LastRunAt); ok {
			row.LastRunAt = &t
		} else {
			row.LastRunAt = nil
		}
	} else {
		row.LastRunAt = nil
	}
	return r.db.Save(row).Error
}

func rowToNotificationData(row model.TenantNotification) dto.NotificationData {
	cfg := dto.NotificationConfig{
		Enabled:             row.Enabled,
		WebhookURL:          row.WebhookURL,
		Secret:              row.Secret,
		PollIntervalMinutes: row.PollIntervalMinutes,
		AppID:               row.AppID,
		AppSecret:           row.AppSecret,
	}
	if row.ScenariosJSON != "" {
		_ = json.Unmarshal([]byte(row.ScenariosJSON), &cfg.Scenarios)
	}
	if row.ShopIDsJSON != "" {
		_ = json.Unmarshal([]byte(row.ShopIDsJSON), &cfg.ShopIDs)
	}
	normalizeNotificationConfig(&cfg)
	state := dto.NotificationState{
		LastRunOK:        row.LastRunOK,
		LastError:        row.LastError,
		LastSentCount:    row.LastSentCount,
		LastBarcodeError: row.LastBarcodeError,
		Notified:         map[string]string{},
	}
	if row.LastRunAt != nil {
		state.LastRunAt = row.LastRunAt.Format(time.RFC3339)
	}
	if row.NotifiedJSON != "" {
		_ = json.Unmarshal([]byte(row.NotifiedJSON), &state.Notified)
	}
	if state.Notified == nil {
		state.Notified = map[string]string{}
	}
	return dto.NotificationData{Config: cfg, State: state}
}

func normalizeNotificationConfig(cfg *dto.NotificationConfig) {
	if cfg.PollIntervalMinutes <= 0 {
		cfg.PollIntervalMinutes = 5
	}
	if cfg.PollIntervalMinutes < 5 {
		cfg.PollIntervalMinutes = 5
	}
}

func pruneNotified(notified map[string]string, keepDays int) {
	if len(notified) == 0 {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -keepDays)
	for key, at := range notified {
		t, ok := parseStoreTime(at)
		if !ok || t.Before(cutoff) {
			delete(notified, key)
		}
	}
	if len(notified) > 10000 {
		for key := range notified {
			if len(notified) <= 8000 {
				break
			}
			delete(notified, key)
		}
	}
}

func parseStoreTime(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05"} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
