package service

import (
	"errors"
	"fmt"
	"strings"

	"aftersalescore/internal/dto"
	"aftersalescore/internal/model"
	"aftersalescore/internal/repo"

	"gorm.io/gorm"
)

func (s *ShopService) shareURL(token string) string {
	base := strings.TrimRight(s.publicBaseURL, "/")
	if base == "" {
		return "/share/returns/" + token
	}
	return base + "/share/returns/" + token
}

func toReturnShareItem(item *model.ReturnShare, shareURL string) dto.ReturnShareItem {
	return dto.ReturnShareItem{
		ID:             item.ID,
		Name:           item.Name,
		Token:          item.Token,
		ReturnLocation: item.ReturnLocation,
		Enabled:        item.Enabled,
		ShareURL:       shareURL,
		CreatedAt:      formatTime(item.CreatedAt),
		UpdatedAt:      formatTime(item.UpdatedAt),
	}
}

func (s *ShopService) ListReturnShares() ([]dto.ReturnShareItem, error) {
	list, err := s.repo().ListReturnShares()
	if err != nil {
		return nil, err
	}
	out := make([]dto.ReturnShareItem, 0, len(list))
	for i := range list {
		out = append(out, toReturnShareItem(&list[i], s.shareURL(list[i].Token)))
	}
	return out, nil
}

func (s *ShopService) CreateReturnShare(in dto.ReturnShareUpsertRequest) (*dto.ReturnShareItem, error) {
	name := strings.TrimSpace(in.Name)
	loc := strings.TrimSpace(in.ReturnLocation)
	if name == "" {
		return nil, fmt.Errorf("%w: 请填写分享者", ErrBadRequest)
	}
	if len(repo.SplitReturnLocationFilters(loc)) == 0 {
		return nil, fmt.Errorf("%w: 请填写退回地过滤", ErrBadRequest)
	}
	token, err := randomHex(16)
	if err != nil {
		return nil, err
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	item := &model.ReturnShare{
		Name:           name,
		Token:          token,
		ReturnLocation: loc,
		Enabled:        enabled,
	}
	if err := s.repo().CreateReturnShare(item); err != nil {
		return nil, err
	}
	out := toReturnShareItem(item, s.shareURL(item.Token))
	return &out, nil
}

func (s *ShopService) UpdateReturnShare(id uint64, in dto.ReturnShareUpsertRequest) (*dto.ReturnShareItem, error) {
	item, err := s.repo().GetReturnShare(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if name := strings.TrimSpace(in.Name); name != "" {
		item.Name = name
	}
	if loc := strings.TrimSpace(in.ReturnLocation); loc != "" {
		if len(repo.SplitReturnLocationFilters(loc)) == 0 {
			return nil, fmt.Errorf("%w: 请填写退回地过滤", ErrBadRequest)
		}
		item.ReturnLocation = loc
	}
	if in.Enabled != nil {
		item.Enabled = *in.Enabled
	}
	if err := s.repo().UpdateReturnShare(item); err != nil {
		return nil, err
	}
	out := toReturnShareItem(item, s.shareURL(item.Token))
	return &out, nil
}

func (s *ShopService) DeleteReturnShare(id uint64) error {
	if _, err := s.repo().GetReturnShare(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	return s.repo().DeleteReturnShare(id)
}

func (s *ShopService) resolvePublicShare(token string) (*ShopService, *model.ReturnShare, error) {
	item, err := s.repos.Shop.GetReturnShareByToken(token)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrNotFound
		}
		return nil, nil, err
	}
	if !item.Enabled {
		return nil, nil, ErrNotFound
	}
	return s.ForTenant(item.TenantID), item, nil
}

func stripShareReturn(item *dto.ReturnPackageItem) {
	item.ShopID = 0
	item.ShopName = ""
	item.PayAmount = ""
	item.RefundAmount = ""
}

func (s *ShopService) PublicReturnShareMeta(token string) (*dto.PublicReturnShareMeta, error) {
	_, item, err := s.resolvePublicShare(token)
	if err != nil {
		return nil, err
	}
	return &dto.PublicReturnShareMeta{
		Name:           item.Name,
		ReturnLocation: item.ReturnLocation,
	}, nil
}

func (s *ShopService) PublicListReturns(token, keyword string, page, pageSize int, sortBy, sortOrder string) (*dto.PublicReturnShareMeta, []dto.ReturnPackageItem, int64, error) {
	svc, item, err := s.resolvePublicShare(token)
	if err != nil {
		return nil, nil, 0, err
	}
	if strings.TrimSpace(sortBy) == "" {
		sortBy = "returnTime"
	}
	if strings.TrimSpace(sortOrder) == "" {
		sortOrder = "desc"
	}
	list, total, err := svc.ListReturns(dto.ReturnListQuery{
		Keyword:        keyword,
		ReturnLocation: item.ReturnLocation,
		Page:           page,
		PageSize:       pageSize,
		SortBy:         sortBy,
		SortOrder:      sortOrder,
	}, "")
	if err != nil {
		return nil, nil, 0, err
	}
	for i := range list {
		stripShareReturn(&list[i])
	}
	return &dto.PublicReturnShareMeta{
		Name:           item.Name,
		ReturnLocation: item.ReturnLocation,
	}, list, total, nil
}

func (s *ShopService) PublicExportReturns(token string, fields []string) ([]byte, string, error) {
	svc, item, err := s.resolvePublicShare(token)
	if err != nil {
		return nil, "", err
	}
	if len(fields) == 0 {
		fields = defaultShareReturnExportFields()
	} else {
		cleaned := make([]string, 0, len(fields))
		for _, f := range fields {
			if strings.TrimSpace(f) == "shop" {
				continue
			}
			cleaned = append(cleaned, f)
		}
		fields = cleaned
	}
	data, filename, err := svc.ExportReturns(dto.ReturnExportRequest{
		Keyword:        "",
		ReturnLocation: item.ReturnLocation,
		Fields:         fields,
		HidePayAmount:  true,
		SortBy:         "returnTime",
		SortOrder:      "desc",
	}, "")
	if err != nil {
		return nil, "", err
	}
	name := strings.TrimSpace(item.Name)
	if name == "" {
		name = "退回件"
	}
	return data, "退回件-" + name + "-" + strings.TrimPrefix(filename, "退回管理-"), nil
}

func defaultShareReturnExportFields() []string {
	out := make([]string, 0, len(returnExportFields))
	for _, f := range returnExportFields {
		if f.Key == "shop" {
			continue
		}
		out = append(out, f.Key)
	}
	return out
}
