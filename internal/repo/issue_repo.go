package repo

import (
	"strings"
	"time"

	"aftersalescore/internal/model"

	"gorm.io/gorm"
)

type IssueRepo struct {
	db       *gorm.DB
	tenantID uint64
}

func NewIssueRepo(db *gorm.DB) *IssueRepo {
	return &IssueRepo{db: db, tenantID: 1}
}

func (r *IssueRepo) ForTenant(tenantID uint64) *IssueRepo {
	return &IssueRepo{db: r.db, tenantID: NormalizeTenantID(tenantID)}
}

type IssueListFilter struct {
	ShopID      uint64
	Status      string
	ProblemType string
	Keyword     string
	Page        int
	PageSize    int
}

func (r *IssueRepo) Create(row *model.AftersaleIssue) error {
	row.TenantID = r.tenantID
	return r.db.Create(row).Error
}

func (r *IssueRepo) Update(row *model.AftersaleIssue) error {
	return r.db.Save(row).Error
}

func (r *IssueRepo) Get(id uint64) (*model.AftersaleIssue, error) {
	var row model.AftersaleIssue
	err := r.db.Scopes(scopeTenant(r.tenantID)).Where("id = ?", id).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *IssueRepo) Delete(id uint64) error {
	return r.db.Scopes(scopeTenant(r.tenantID)).Where("id = ?", id).Delete(&model.AftersaleIssue{}).Error
}

func (r *IssueRepo) List(f IssueListFilter) ([]model.AftersaleIssue, int64, error) {
	q := r.db.Model(&model.AftersaleIssue{}).Scopes(scopeTenant(r.tenantID))
	if f.ShopID > 0 {
		q = q.Where("shop_id = ?", f.ShopID)
	}
	if s := strings.TrimSpace(f.Status); s != "" {
		q = q.Where("status = ?", s)
	}
	if p := strings.TrimSpace(f.ProblemType); p != "" {
		q = q.Where("problem_type = ?", p)
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where(
			"order_no ILIKE ? OR platform_order_id ILIKE ? OR platform_aftersale_id ILIKE ? OR product_title ILIKE ? OR shop_name ILIKE ? OR sku_specs ILIKE ? OR buyer_name ILIKE ? OR buyer_phone ILIKE ? OR problem_note ILIKE ?",
			like, like, like, like, like, like, like, like, like,
		)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := f.Page, f.PageSize
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 200 {
		size = 20
	}
	var list []model.AftersaleIssue
	err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&list).Error
	return list, total, err
}

// CountPendingRecent 最近 days 天内创建的待处理问题单数量。
func (r *IssueRepo) CountPendingRecent(days int) (int64, error) {
	if days <= 0 {
		days = 7
	}
	since := time.Now().AddDate(0, 0, -days)
	var n int64
	err := r.db.Model(&model.AftersaleIssue{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("status = ? AND created_at >= ?", model.IssueStatusPending, since).
		Count(&n).Error
	return n, err
}
