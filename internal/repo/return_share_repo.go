package repo

import (
	"strings"
	"unicode"

	"aftersalescore/internal/model"

	"gorm.io/gorm"
)

func SplitReturnLocationFilters(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '，' || r == ';' || r == '；' || r == '\n' || r == '\r' || r == '|'
	})
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.TrimFunc(p, unicode.IsSpace)
		if p == "" {
			continue
		}
		key := strings.ToLower(p)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, p)
	}
	return out
}

func (r *ShopRepo) ListReturnShares() ([]model.ReturnShare, error) {
	var list []model.ReturnShare
	err := r.db.Scopes(scopeTenant(r.tenantID)).Order("id DESC").Find(&list).Error
	return list, err
}

func (r *ShopRepo) GetReturnShare(id uint64) (*model.ReturnShare, error) {
	var item model.ReturnShare
	err := r.db.Scopes(scopeTenant(r.tenantID)).First(&item, id).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ShopRepo) GetReturnShareByToken(token string) (*model.ReturnShare, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var item model.ReturnShare
	err := r.db.Where("token = ?", token).First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ShopRepo) CreateReturnShare(item *model.ReturnShare) error {
	item.TenantID = r.tenantID
	return r.db.Create(item).Error
}

func (r *ShopRepo) UpdateReturnShare(item *model.ReturnShare) error {
	return r.db.Scopes(scopeTenant(r.tenantID)).Save(item).Error
}

func (r *ShopRepo) DeleteReturnShare(id uint64) error {
	return r.db.Scopes(scopeTenant(r.tenantID)).Delete(&model.ReturnShare{}, id).Error
}
