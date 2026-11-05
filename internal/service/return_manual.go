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

	"gorm.io/gorm"
)

func (s *ShopService) UploadReturnImage(file *multipart.FileHeader) (string, error) {
	if s.store == nil {
		return "", fmt.Errorf("%w: 存储未配置", ErrBadRequest)
	}
	if file == nil {
		return "", fmt.Errorf("%w: 请选择图片", ErrBadRequest)
	}
	if file.Size > 10<<20 {
		return "", fmt.Errorf("%w: 图片不能超过 10MB", ErrBadRequest)
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp":
	default:
		ct := strings.ToLower(file.Header.Get("Content-Type"))
		if !strings.HasPrefix(ct, "image/") {
			return "", fmt.Errorf("%w: 请上传图片文件", ErrBadRequest)
		}
	}
	_, url, err := s.store.Upload(file, "returns")
	if err != nil {
		return "", err
	}
	return url, nil
}

func (s *ShopService) CreateManualReturn(in dto.ManualReturnRequest, bearerToken string) (*dto.ReturnPackageItem, error) {
	shopName := strings.TrimSpace(in.ShopName)
	var shopID uint64
	if in.ShopID > 0 {
		shop, err := s.repo().Get(in.ShopID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("%w: 店铺不存在", ErrBadRequest)
			}
			return nil, err
		}
		shopID = shop.ID
		if shopName == "" {
			shopName = shop.Name
		}
	}
	if shopName == "" {
		return nil, fmt.Errorf("%w: 请填写店铺名", ErrBadRequest)
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
	if err := s.ensureReturnAftersaleFree(shopID, aid, 0); err != nil {
		return nil, err
	}
	now := time.Now()
	item := &model.ReturnPackage{
		Status:   "已退回",
		RawJSON:  `{"source":"manual"}`,
		SyncedAt: now,
	}
	if err := applyManualReturnFields(item, in, shopID, shopName, aid, orderNo, logisticsNo, title); err != nil {
		return nil, err
	}
	if err := s.repo().CreateReturn(item); err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("%w: 该售后编号已存在", ErrBadRequest)
		}
		return nil, err
	}
	out := toReturnItem(item, shopName)
	list := []dto.ReturnPackageItem{out}
	s.attachFenFaRemarks(list, bearerToken)
	return &list[0], nil
}

func isManualReturn(item *model.ReturnPackage) bool {
	return item != nil && strings.Contains(item.RawJSON, `"source":"manual"`)
}

func (s *ShopService) ensureReturnAftersaleFree(shopID uint64, aftersaleID string, exceptID uint64) error {
	existing, err := s.repo().GetReturnByShopAftersale(shopID, aftersaleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if existing != nil && existing.ID != exceptID {
		return fmt.Errorf("%w: 该售后编号已存在", ErrBadRequest)
	}
	return nil
}

func applyManualReturnFields(item *model.ReturnPackage, in dto.ManualReturnRequest, shopID uint64, shopName, aid, orderNo, logisticsNo, title string) error {
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
	item.ShopID = shopID
	item.ShopName = shopName
	item.PlatformAftersaleID = aid
	item.OrderNo = orderNo
	item.ProductTitle = title
	item.ProductImage = strings.TrimSpace(in.ProductImage)
	item.SKU = strings.TrimSpace(in.SKU)
	item.Qty = qty
	item.BuyQty = buyQty
	item.PayAmount = strings.TrimSpace(in.PayAmount)
	item.RefundAmount = strings.TrimSpace(in.RefundAmount)
	item.AftersaleType = typ
	item.Reason = strings.TrimSpace(in.Reason)
	item.LogisticsNo = logisticsNo
	item.Carrier = strings.TrimSpace(in.Carrier)
	item.ReturnLocation = strings.TrimSpace(in.ReturnLocation)
	item.FenFaRemark = strings.TrimSpace(in.FenFaRemark)
	item.ApplyTime = applyTime
	item.ReturnTime = returnTime
	item.ReturnedAt = returnedAt
	item.AppliedAt = appliedAt
	item.SyncedAt = time.Now()
	return nil
}

func (s *ShopService) UpdateManualReturn(id uint64, in dto.ManualReturnRequest, bearerToken string) (*dto.ReturnPackageItem, error) {
	item, err := s.repo().GetReturn(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if !isManualReturn(item) {
		return nil, fmt.Errorf("%w: 仅手工添加的退回件可编辑", ErrBadRequest)
	}
	shopName := strings.TrimSpace(in.ShopName)
	var shopID uint64
	if in.ShopID > 0 {
		shop, err := s.repo().Get(in.ShopID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("%w: 店铺不存在", ErrBadRequest)
			}
			return nil, err
		}
		shopID = shop.ID
		if shopName == "" {
			shopName = shop.Name
		}
	} else {
		shopID = item.ShopID
	}
	if shopName == "" {
		return nil, fmt.Errorf("%w: 请填写店铺名", ErrBadRequest)
	}
	orderNo := strings.TrimSpace(in.OrderNo)
	logisticsNo := strings.TrimSpace(in.LogisticsNo)
	aid := strings.TrimSpace(in.PlatformAftersaleID)
	title := strings.TrimSpace(in.ProductTitle)
	if orderNo == "" && logisticsNo == "" && aid == "" {
		return nil, fmt.Errorf("%w: 请填写订单号、售后编号或物流单号", ErrBadRequest)
	}
	if aid == "" {
		aid = strings.TrimSpace(item.PlatformAftersaleID)
	}
	if err := s.ensureReturnAftersaleFree(shopID, aid, item.ID); err != nil {
		return nil, err
	}
	if err := applyManualReturnFields(item, in, shopID, shopName, aid, orderNo, logisticsNo, title); err != nil {
		return nil, err
	}
	if strings.TrimSpace(item.RawJSON) == "" || !isManualReturn(item) {
		item.RawJSON = `{"source":"manual"}`
	}
	if err := s.repo().UpdateReturn(item); err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("%w: 该售后编号已存在", ErrBadRequest)
		}
		return nil, err
	}
	out := toReturnItem(item, shopName)
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
