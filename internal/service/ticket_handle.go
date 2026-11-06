package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"aftersalescore/internal/dto"
	"aftersalescore/internal/model"

	"gorm.io/gorm"
)

func NormalizeTicketHandleStatus(v string) string {
	switch strings.TrimSpace(v) {
	case "", model.TicketHandlePending, "未处理":
		return model.TicketHandlePending
	case model.TicketHandleHandled, "已处理":
		return model.TicketHandleHandled
	case model.TicketHandleIssue, "已转问题记录":
		return model.TicketHandleIssue
	default:
		return ""
	}
}

func TicketHandleStatusLabel(v string) string {
	switch NormalizeTicketHandleStatus(v) {
	case model.TicketHandleHandled:
		return "已处理"
	case model.TicketHandleIssue:
		return "已转问题记录"
	default:
		return "未处理"
	}
}

func ticketHandleStatusOf(t *model.AftersaleTicket) string {
	if t == nil {
		return model.TicketHandlePending
	}
	return NormalizeTicketHandleStatus(t.HandleStatus)
}

func matchTicketHandleStatus(t *model.AftersaleTicket, want string) bool {
	raw := strings.TrimSpace(want)
	if raw == "" || raw == "all" || raw == "全部" {
		return true
	}
	normalized := NormalizeTicketHandleStatus(raw)
	if normalized == "" {
		return true
	}
	return ticketHandleStatusOf(t) == normalized
}

func (s *ShopService) SetTicketHandleStatus(id uint64, status string, operatorID uint64, operatorName string) (*dto.TicketItem, error) {
	st := NormalizeTicketHandleStatus(status)
	if st == "" || (st != model.TicketHandlePending && st != model.TicketHandleHandled) {
		return nil, fmt.Errorf("%w: 无效处理状态", ErrBadRequest)
	}
	item, err := s.repo().GetTicket(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if ticketHandleStatusOf(item) == model.TicketHandleIssue && st == model.TicketHandleHandled {
		return nil, fmt.Errorf("%w: 已转问题记录，无需再标记已处理", ErrBadRequest)
	}
	now := time.Now()
	var handledAt *time.Time
	if st == model.TicketHandleHandled {
		handledAt = &now
	}
	if err := s.repo().UpdateTicketHandle(id, st, operatorID, strings.TrimSpace(operatorName), handledAt); err != nil {
		return nil, err
	}
	item, err = s.repo().GetTicket(id)
	if err != nil {
		return nil, err
	}
	out := toTicketItem(item)
	return &out, nil
}

func (s *ShopService) TransferTicketToIssue(id uint64, in dto.IssueUpsertRequest, operatorID uint64, operatorName string) (*dto.IssueItem, error) {
	item, err := s.repo().GetTicket(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	fillIssueFromTicket(&in, item)
	if strings.TrimSpace(in.ShopName) == "" && item.ShopID > 0 {
		if shop, err := s.repo().Get(item.ShopID); err == nil {
			in.ShopName = shop.Name
			if in.ShopID == 0 {
				in.ShopID = shop.ID
			}
		}
	}
	issue, err := s.CreateIssue(in, operatorID, operatorName)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if err := s.repo().UpdateTicketHandle(id, model.TicketHandleIssue, operatorID, strings.TrimSpace(operatorName), &now); err != nil {
		return nil, err
	}
	return issue, nil
}

func fillIssueFromTicket(in *dto.IssueUpsertRequest, t *model.AftersaleTicket) {
	if in.ShopID == 0 {
		in.ShopID = t.ShopID
	}
	if strings.TrimSpace(in.OrderNo) == "" {
		in.OrderNo = strings.TrimSpace(t.OrderNo)
	}
	if strings.TrimSpace(in.PlatformAftersaleID) == "" {
		in.PlatformAftersaleID = strings.TrimSpace(t.PlatformAftersaleID)
	}
	if strings.TrimSpace(in.ProductTitle) == "" {
		in.ProductTitle = strings.TrimSpace(t.ProductTitle)
	}
	if strings.TrimSpace(in.ProductImage) == "" {
		in.ProductImage = strings.TrimSpace(t.ProductImage)
	}
	if strings.TrimSpace(in.SkuSpecs) == "" {
		in.SkuSpecs = strings.TrimSpace(t.SKU)
	}
	in.HasAftersale = true
	if strings.TrimSpace(in.AftersaleStatus) == "" {
		in.AftersaleStatus = strings.TrimSpace(t.Status)
	}
	if strings.TrimSpace(in.AftersaleType) == "" {
		in.AftersaleType = strings.TrimSpace(t.AftersaleType)
	}
	if strings.TrimSpace(in.Logistics) == "" {
		in.Logistics = strings.TrimSpace(t.Logistics)
	}
	if strings.TrimSpace(in.ReturnLogisticsNo) == "" {
		in.ReturnLogisticsNo = strings.TrimSpace(t.ReturnLogisticsNo)
	}
	if strings.TrimSpace(in.ShipLogisticsNo) == "" {
		in.ShipLogisticsNo = strings.TrimSpace(t.ShipLogisticsNo)
	}
}
