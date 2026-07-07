package repo

import (
	"strings"

	"aftersalescore/internal/model"

	"gorm.io/gorm"
)

type EdgeRecordListFilter struct {
	Type       string
	TrackingNo string
	EdgeID     string
	Status     string
	Page       int
	PageSize   int
}

type EdgeRecordRepo struct {
	db *gorm.DB
}

func NewEdgeRecordRepo(db *gorm.DB) *EdgeRecordRepo {
	return &EdgeRecordRepo{db: db}
}

func (r *EdgeRecordRepo) List(f EdgeRecordListFilter) ([]model.EdgeRecord, int64, error) {
	q := r.db.Model(&model.EdgeRecord{}).Where("status = ?", model.EdgeRecordStatusCompleted)
	if t := strings.TrimSpace(f.Type); t != "" {
		q = q.Where("type = ?", t)
	}
	if tn := NormalizeTrackingNo(f.TrackingNo); tn != "" {
		q = q.Where("tracking_number ILIKE ?", "%"+tn+"%")
	}
	if eid := strings.TrimSpace(f.EdgeID); eid != "" {
		q = q.Where("edge_id = ?", eid)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, pageSize := f.Page, f.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	var list []model.EdgeRecord
	err := q.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&list).Error
	return list, total, err
}

func (r *EdgeRecordRepo) Get(id int64) (*model.EdgeRecord, error) {
	var rec model.EdgeRecord
	err := r.db.First(&rec, id).Error
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *EdgeRecordRepo) Create(rec *model.EdgeRecord) error {
	return r.db.Create(rec).Error
}

func (r *EdgeRecordRepo) Save(rec *model.EdgeRecord) error {
	return r.db.Save(rec).Error
}

func (r *EdgeRecordRepo) Delete(id int64) error {
	return r.db.Delete(&model.EdgeRecord{}, id).Error
}

func (r *EdgeRecordRepo) DeleteBatch(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Where("id IN ?", ids).Delete(&model.EdgeRecord{}).Error
}

func (r *EdgeRecordRepo) DistinctEdgeIDs() ([]string, error) {
	var ids []string
	err := r.db.Model(&model.EdgeRecord{}).
		Where("edge_id <> ''").
		Distinct("edge_id").
		Pluck("edge_id", &ids).Error
	return ids, err
}

func (r *EdgeRecordRepo) CountByType(recordType string) (int64, error) {
	var n int64
	err := r.db.Model(&model.EdgeRecord{}).
		Where("type = ? AND status = ?", recordType, model.EdgeRecordStatusCompleted).
		Count(&n).Error
	return n, err
}
