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

func (s *ShopService) CreateManualReturn(in dto.ManualReturnRequest, bearerToken string) (*dto.ReturnPackageItem, error) {
	if in.ShopID == 0 {
		return nil, fmt.Errorf("%w: 请选择店铺", ErrBadRequest)
	}
	shop, err := s.repo().Get(in.ShopID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: 店铺不存在", ErrBadRequest)
		}
		return nil, err
	}
	orderNo := strings.TrimSpace(in.OrderNo)
	logisticsNo := strings.TrimSpace(in.LogisticsNo)
	aid := strings.TrimSpace(in.PlatformAftersaleID)
	title := strings.TrimSpace(in.ProductTitle)
	if orderNo == "" && logisticsNo == "" && aid == "" {
		return nil, fmt.Errorf("%w: 请填写订单号、售后编号或物流单号", ErrBadRequest)
	}
	if aid == "" {
		hex, err := randomHex(3)
		if err != nil {
			return nil, err
		}
		aid = "MANUAL-" + time.Now().Format("20060102150405") + strings.ToUpper(hex)
	}
	if existing, err := s.repo().GetReturnByShopAftersale(shop.ID, aid); err == nil && existing != nil {
		return nil, fmt.Errorf("%w: 该售后编号已存在", ErrBadRequest)
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	applyTime := strings.TrimSpace(in.ApplyTime)
	returnTime, returnedAt, appliedAt := ResolveReturnTimes(strings.TrimSpace(in.ReturnTime), applyTime, "", "")
	qty := in.Qty
	if qty < 0 {
		qty = 0
	}
	buyQty := in.BuyQty
	if buyQty <= 0 {
		buyQty = qty
	}
	typ := strings.TrimSpace(in.AftersaleType)
	if typ == "" {
		typ = "已发货退款"
	}
	now := time.Now()
	item := &model.ReturnPackage{
		ShopID:              shop.ID,
		PlatformAftersaleID: aid,
		OrderNo:             orderNo,
		ProductTitle:        title,
		ProductImage:        strings.TrimSpace(in.ProductImage),
		SKU:                 strings.TrimSpace(in.SKU),
		Qty:                 qty,
		BuyQty:              buyQty,
		PayAmount:           strings.TrimSpace(in.PayAmount),
		RefundAmount:        strings.TrimSpace(in.RefundAmount),
		AftersaleType:       typ,
		Reason:              strings.TrimSpace(in.Reason),
		Status:              "已退回",
		LogisticsNo:         logisticsNo,
		Carrier:             strings.TrimSpace(in.Carrier),
		ReturnLocation:      strings.TrimSpace(in.ReturnLocation),
		ApplyTime:           applyTime,
		ReturnTime:          returnTime,
		ReturnedAt:          returnedAt,
		AppliedAt:           appliedAt,
		RawJSON:             `{"source":"manual"}`,
		SyncedAt:            now,
	}
	if err := s.repo().CreateReturn(item); err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("%w: 该售后编号已存在", ErrBadRequest)
		}
		return nil, err
	}
	out := toReturnItem(item, shop.Name)
	list := []dto.ReturnPackageItem{out}
	s.attachFenFaRemarks(list, bearerToken)
	return &list[0], nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique")
}
