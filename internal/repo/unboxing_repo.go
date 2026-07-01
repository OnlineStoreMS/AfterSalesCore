package repo

import (
	"strings"

	"aftersalescore/internal/model"

	"gorm.io/gorm"
)

type UnboxingRepo struct {
	db       *gorm.DB
	tenantID uint64
}

func NewUnboxingRepo(db *gorm.DB) *UnboxingRepo {
	return &UnboxingRepo{db: db}
}

func (r *UnboxingRepo) ForTenant(tenantID uint64) *UnboxingRepo {
	return &UnboxingRepo{db: r.db, tenantID: NormalizeTenantID(tenantID)}
}

type UnboxingListFilter struct {
	TrackingNo string
	Page       int
	PageSize   int
}

func NormalizeTrackingNo(v string) string {
	return strings.ToUpper(strings.TrimSpace(v))
}

func (r *UnboxingRepo) List(f UnboxingListFilter) ([]model.UnboxingRecord, int64, error) {
	q := r.db.Model(&model.UnboxingRecord{}).Scopes(scopeTenant(r.tenantID))
	if f.TrackingNo != "" {
		like := "%" + NormalizeTrackingNo(f.TrackingNo) + "%"
		q = q.Where("tracking_no ILIKE ?", like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.UnboxingRecord
	offset := (f.Page - 1) * f.PageSize
	err := q.Order("created_at DESC").Offset(offset).Limit(f.PageSize).Find(&list).Error
	return list, total, err
}

func (r *UnboxingRepo) GetWithPhotos(id uint64) (*model.UnboxingRecord, error) {
	var rec model.UnboxingRecord
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Preload("Photos", func(db *gorm.DB) *gorm.DB { return db.Order("sort_order ASC, id ASC") }).
		First(&rec, id).Error
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *UnboxingRepo) Create(rec *model.UnboxingRecord) error {
	rec.TenantID = r.tenantID
	return r.db.Create(rec).Error
}

func (r *UnboxingRepo) Save(rec *model.UnboxingRecord) error {
	return r.db.Save(rec).Error
}

func (r *UnboxingRepo) AddPhoto(photo *model.UnboxingPhoto) error {
	photo.TenantID = r.tenantID
	return r.db.Create(photo).Error
}

func (r *UnboxingRepo) CountPhotos(recordID uint64) (int64, error) {
	var n int64
	err := r.db.Model(&model.UnboxingPhoto{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("record_id = ?", recordID).
		Count(&n).Error
	return n, err
}

func (r *UnboxingRepo) NextPhotoSort(recordID uint64) (int, error) {
	var max *int
	err := r.db.Model(&model.UnboxingPhoto{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("record_id = ?", recordID).
		Select("MAX(sort_order)").Scan(&max).Error
	if err != nil {
		return 0, err
	}
	if max == nil {
		return 0, nil
	}
	return *max + 1, nil
}
