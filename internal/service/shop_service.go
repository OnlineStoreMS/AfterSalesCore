package service

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"aftersalescore/internal/dto"
	"aftersalescore/internal/model"
	"aftersalescore/internal/repo"

	"gorm.io/gorm"
)

const (
	pluginOnlineSkew = 3 * time.Minute
)

var (
	ErrPluginAuth      = errors.New("插件鉴权失败")
	ErrAlreadyBound    = errors.New("店铺已绑定插件，请先重置绑定码")
	ErrBindCodeInvalid = errors.New("绑定码无效")
)

var shopPlatforms = map[string]string{
	model.ShopPlatformDoudian:   "抖店",
	model.ShopPlatformTaobao:    "淘宝",
	model.ShopPlatformPinduoduo: "拼多多",
}

type ShopService struct {
	repos    *repo.Repos
	tenantID uint64
}

func NewShopService(repos *repo.Repos) *ShopService {
	return &ShopService{repos: repos}
}

func (s *ShopService) ForTenant(tenantID uint64) *ShopService {
	return &ShopService{repos: s.repos, tenantID: repo.NormalizeTenantID(tenantID)}
}

func (s *ShopService) repo() *repo.ShopRepo {
	return s.repos.Shop.ForTenant(s.tenantID)
}

func PlatformLabel(platform string) string {
	if v, ok := shopPlatforms[platform]; ok {
		return v
	}
	return platform
}

func PluginAvailable(platform string) bool {
	return platform == model.ShopPlatformDoudian
}

func (s *ShopService) List() ([]dto.ShopItem, error) {
	list, err := s.repo().List()
	if err != nil {
		return nil, err
	}
	counts, err := s.repo().CountPendingTickets()
	if err != nil {
		return nil, err
	}
	out := make([]dto.ShopItem, 0, len(list))
	for i := range list {
		item := s.toItem(&list[i])
		item.PendingTicketCount = counts[list[i].ID]
		out = append(out, item)
	}
	return out, nil
}

func (s *ShopService) Get(id uint64) (*dto.ShopItem, error) {
	shop, err := s.repo().Get(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	item := s.toItem(shop)
	counts, err := s.repo().CountPendingTickets()
	if err != nil {
		return nil, err
	}
	item.PendingTicketCount = counts[shop.ID]
	return &item, nil
}

func (s *ShopService) Create(in *dto.ShopCreateInput) (*dto.ShopItem, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, ErrBadRequest
	}
	platform := strings.TrimSpace(in.Platform)
	if platform == "" {
		platform = model.ShopPlatformDoudian
	}
	if _, ok := shopPlatforms[platform]; !ok {
		return nil, fmt.Errorf("%w: 不支持的店铺平台", ErrBadRequest)
	}
	var shop *model.MarketplaceShop
	var lastErr error
	for i := 0; i < 6; i++ {
		code, err := randomBindCode()
		if err != nil {
			return nil, err
		}
		shop = &model.MarketplaceShop{
			Name:         name,
			Platform:     platform,
			BindCode:     code,
			PluginStatus: model.ShopPluginUnbound,
			Remark:       strings.TrimSpace(in.Remark),
		}
		lastErr = s.repo().Create(shop)
		if lastErr == nil {
			item := s.toItem(shop)
			return &item, nil
		}
		if !strings.Contains(strings.ToLower(lastErr.Error()), "unique") &&
			!strings.Contains(strings.ToLower(lastErr.Error()), "duplicate") {
			return nil, lastErr
		}
	}
	return nil, lastErr
}

func (s *ShopService) Update(id uint64, in *dto.ShopUpdateInput) (*dto.ShopItem, error) {
	shop, err := s.repo().Get(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if name := strings.TrimSpace(in.Name); name != "" {
		shop.Name = name
	}
	shop.Remark = strings.TrimSpace(in.Remark)
	if err := s.repo().Save(shop); err != nil {
		return nil, err
	}
	item := s.toItem(shop)
	return &item, nil
}

func (s *ShopService) Delete(id uint64) error {
	shop, err := s.repo().Get(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	return s.repos.Shop.ForTenant(shop.TenantID).Delete(shop.ID)
}

func (s *ShopService) ResetBind(id uint64) (*dto.ShopItem, error) {
	shop, err := s.repo().Get(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	code, err := randomBindCode()
	if err != nil {
		return nil, err
	}
	shop.BindCode = code
	shop.PluginKey = ""
	shop.PluginSecretHash = ""
	shop.PluginStatus = model.ShopPluginUnbound
	shop.LastSeenAt = nil
	shop.SyncRequestedAt = nil
	if err := s.repo().Save(shop); err != nil {
		return nil, err
	}
	item := s.toItem(shop)
	return &item, nil
}

func (s *ShopService) Workbench(id uint64, cardKey, keyword string, page, pageSize int) (*dto.ShopWorkbench, error) {
	shop, err := s.repo().Get(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	cards, err := s.repo().ListCards(shop.ID)
	if err != nil {
		return nil, err
	}
	tickets, total, err := s.repo().ListTickets(repo.TicketListFilter{
		ShopID: shop.ID, CardKey: cardKey, Keyword: keyword, Page: page, PageSize: pageSize,
	})
	if err != nil {
		return nil, err
	}
	item := s.toItem(shop)
	out := &dto.ShopWorkbench{
		Shop:     item,
		Cards:    make([]dto.FilterCardItem, 0, len(cards)),
		Tickets:  make([]dto.TicketItem, 0, len(tickets)),
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}
	if shop.LastSyncAt != nil {
		out.LastSyncAt = formatTime(*shop.LastSyncAt)
	}
	for _, c := range cards {
		out.Cards = append(out.Cards, dto.FilterCardItem{
			GroupName: c.GroupName, CardKey: c.CardKey, CardLabel: c.CardLabel,
			Count: c.Count, SortOrder: c.SortOrder,
		})
	}
	for i := range tickets {
		out.Tickets = append(out.Tickets, toTicketItem(&tickets[i]))
	}
	return out, nil
}

func (s *ShopService) ListTickets(id uint64, cardKey, keyword string, page, pageSize int) ([]dto.TicketItem, int64, error) {
	shop, err := s.repo().Get(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, 0, ErrNotFound
	}
	if err != nil {
		return nil, 0, err
	}
	list, total, err := s.repo().ListTickets(repo.TicketListFilter{
		ShopID: shop.ID, CardKey: cardKey, Keyword: keyword, Page: page, PageSize: pageSize,
	})
	if err != nil {
		return nil, 0, err
	}
	out := make([]dto.TicketItem, 0, len(list))
	for i := range list {
		out = append(out, toTicketItem(&list[i]))
	}
	return out, total, nil
}

func (s *ShopService) ListReturns(q dto.ReturnListQuery) ([]dto.ReturnPackageItem, int64, error) {
	if q.ShopID > 0 {
		if _, err := s.repo().Get(q.ShopID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, 0, ErrNotFound
			}
			return nil, 0, err
		}
	}
	list, total, err := s.repo().ListReturns(repo.ReturnListFilter{
		ShopID:     q.ShopID,
		Keyword:    q.Keyword,
		ReturnFrom: ParseQueryDateTime(q.ReturnFrom, false),
		ReturnTo:   ParseQueryDateTime(q.ReturnTo, true),
		ApplyFrom:  ParseQueryDateTime(q.ApplyFrom, false),
		ApplyTo:    ParseQueryDateTime(q.ApplyTo, true),
		Page:       q.Page,
		PageSize:   q.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}
	shops, err := s.repo().List()
	if err != nil {
		return nil, 0, err
	}
	names := make(map[uint64]string, len(shops))
	for i := range shops {
		names[shops[i].ID] = shops[i].Name
	}
	out := make([]dto.ReturnPackageItem, 0, len(list))
	for i := range list {
		out = append(out, toReturnItem(&list[i], names[list[i].ShopID]))
	}
	return out, total, nil
}

func interceptKey(shopID uint64, aftersaleID string) string {
	return strconv.FormatUint(shopID, 10) + ":" + strings.TrimSpace(aftersaleID)
}

func (s *ShopService) ListIntercepts(q dto.InterceptListQuery) ([]dto.InterceptItem, int64, error) {
	if q.ShopID > 0 {
		if _, err := s.repo().Get(q.ShopID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, 0, ErrNotFound
			}
			return nil, 0, err
		}
	}
	tickets, err := s.repo().ListInterceptTickets(q.ShopID, q.Keyword)
	if err != nil {
		return nil, 0, err
	}
	pickups, err := s.repo().ListShippedRefundsByStatus(q.ShopID, LogisticsAwaitPickup)
	if err != nil {
		return nil, 0, err
	}
	shops, err := s.repo().List()
	if err != nil {
		return nil, 0, err
	}
	names := make(map[uint64]string, len(shops))
	for i := range shops {
		names[shops[i].ID] = shops[i].Name
	}
	kw := strings.TrimSpace(q.Keyword)
	merged := map[string]*dto.InterceptItem{}
	order := make([]string, 0)
	for i := range tickets {
		t := &tickets[i]
		key := interceptKey(t.ShopID, t.PlatformAftersaleID)
		item := interceptFromTicket(t, names[t.ShopID])
		merged[key] = &item
		order = append(order, key)
	}
	for i := range pickups {
		p := &pickups[i]
		if !IsMerchantInterceptPickup(p.Logistics, p.LogisticsStatus) {
			continue
		}
		if kw != "" && !interceptKeywordMatch(p, kw) {
			continue
		}
		key := interceptKey(p.ShopID, p.PlatformAftersaleID)
		if existing := merged[key]; existing != nil {
			existing.AwaitPickup = true
			existing.LogisticsStatus = LogisticsAwaitPickup
			if existing.LogisticsNo == "" {
				existing.LogisticsNo = strings.TrimSpace(p.LogisticsNo)
			}
			if existing.Carrier == "" {
				existing.Carrier = strings.TrimSpace(p.Carrier)
			}
			existing.Source = "both"
			continue
		}
		item := interceptFromShipped(p, names[p.ShopID])
		merged[key] = &item
		order = append(order, key)
	}
	list := make([]dto.InterceptItem, 0, len(order))
	for _, key := range order {
		if item := merged[key]; item != nil {
			list = append(list, *item)
		}
	}
	sort.SliceStable(list, func(i, j int) bool {
		ti := interceptSortTime(list[i])
		tj := interceptSortTime(list[j])
		if !ti.Equal(tj) {
			return ti.After(tj)
		}
		return list[i].ID > list[j].ID
	})
	total := int64(len(list))
	page, pageSize := q.Page, q.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	start := (page - 1) * pageSize
	if start >= len(list) {
		return []dto.InterceptItem{}, total, nil
	}
	end := start + pageSize
	if end > len(list) {
		end = len(list)
	}
	return list[start:end], total, nil
}

func interceptKeywordMatch(p *model.ShippedRefundSuccess, kw string) bool {
	blob := strings.ToLower(strings.Join([]string{
		p.PlatformAftersaleID, p.OrderNo, p.ProductTitle, p.SKU, p.Logistics, p.LogisticsNo, p.Status, p.OrderInfo,
	}, " "))
	return strings.Contains(blob, strings.ToLower(kw))
}

func interceptSortTime(item dto.InterceptItem) time.Time {
	if t := ParsePlatformDateTime(item.ApplyTime, 0); t != nil {
		return *t
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", item.SyncedAt, time.Local); err == nil {
		return t
	}
	return time.Time{}
}

func interceptFromTicket(t *model.AftersaleTicket, shopName string) dto.InterceptItem {
	no := strings.TrimSpace(t.ShipLogisticsNo)
	if no == "" {
		no = strings.TrimSpace(t.ReturnLogisticsNo)
	}
	return dto.InterceptItem{
		ID: t.ID, ShopID: t.ShopID, ShopName: shopName, Source: "intercept",
		NeedIntercept: true, AwaitPickup: false,
		PlatformAftersaleID: t.PlatformAftersaleID, OrderNo: t.OrderNo,
		ProductTitle: t.ProductTitle, ProductImage: t.ProductImage, SKU: t.SKU,
		Qty: t.Qty, BuyQty: t.BuyQty, PayAmount: t.PayAmount, RefundAmount: t.RefundAmount,
		AftersaleType: t.AftersaleType, Reason: t.Reason, Status: t.Status,
		Logistics: t.Logistics, LogisticsNo: no, ShipLogisticsNo: t.ShipLogisticsNo,
		ReturnLogisticsNo: t.ReturnLogisticsNo, ApplyTime: t.ApplyTime, SyncedAt: formatTime(t.SyncedAt),
	}
}

func interceptFromShipped(p *model.ShippedRefundSuccess, shopName string) dto.InterceptItem {
	status := ClassifyShippedRefundStatus(p.Logistics, p.TrackJSON, p.LogisticsStatus)
	return dto.InterceptItem{
		ID: p.ID, ShopID: p.ShopID, ShopName: shopName, Source: "pickup",
		NeedIntercept: false, AwaitPickup: true,
		PlatformAftersaleID: p.PlatformAftersaleID, OrderNo: p.OrderNo,
		ProductTitle: p.ProductTitle, ProductImage: p.ProductImage, SKU: p.SKU,
		Qty: p.Qty, BuyQty: p.BuyQty, PayAmount: p.PayAmount, RefundAmount: p.RefundAmount,
		AftersaleType: p.AftersaleType, Reason: p.Reason, Status: p.Status,
		Logistics: p.Logistics, LogisticsStatus: firstNonEmpty(status, LogisticsAwaitPickup),
		LogisticsNo: p.LogisticsNo, Carrier: p.Carrier, ApplyTime: p.ApplyTime, SyncedAt: formatTime(p.SyncedAt),
	}
}

func (s *ShopService) ListShippedRefunds(q dto.ShippedRefundListQuery) ([]dto.ShippedRefundItem, int64, error) {
	if q.ShopID > 0 {
		if _, err := s.repo().Get(q.ShopID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, 0, ErrNotFound
			}
			return nil, 0, err
		}
	}
	list, _, err := s.repo().ListShippedRefunds(repo.ShippedRefundListFilter{
		ShopID:    q.ShopID,
		Keyword:   q.Keyword,
		ApplyFrom: ParseQueryDateTime(q.ApplyFrom, false),
		ApplyTo:   ParseQueryDateTime(q.ApplyTo, true),
		Unpaged:   true,
	})
	if err != nil {
		return nil, 0, err
	}
	shops, err := s.repo().List()
	if err != nil {
		return nil, 0, err
	}
	names := make(map[uint64]string, len(shops))
	for i := range shops {
		names[shops[i].ID] = shops[i].Name
	}
	wantStatus := strings.TrimSpace(q.Status)
	out := make([]dto.ShippedRefundItem, 0, len(list))
	for i := range list {
		item := toShippedRefundItem(&list[i], names[list[i].ShopID])
		if wantStatus != "" && item.LogisticsStatus != wantStatus {
			continue
		}
		if q.AlertOnly && !item.Alert {
			continue
		}
		out = append(out, item)
	}
	total := int64(len(out))
	page, pageSize := q.Page, q.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	start := (page - 1) * pageSize
	if start >= len(out) {
		return []dto.ShippedRefundItem{}, total, nil
	}
	end := start + pageSize
	if end > len(out) {
		end = len(out)
	}
	return out[start:end], total, nil
}

func (s *ShopService) Bind(bindCode string) (*dto.PluginBindResult, error) {
	code := strings.ToUpper(strings.TrimSpace(bindCode))
	if len(code) < 4 {
		return nil, ErrBindCodeInvalid
	}
	shop, err := s.repos.Shop.GetByBindCode(code)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrBindCodeInvalid
	}
	if err != nil {
		return nil, err
	}
	if shop.PluginKey != "" {
		return nil, ErrAlreadyBound
	}
	key, err := randomHex(16)
	if err != nil {
		return nil, err
	}
	secret, err := randomHex(24)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	shop.PluginKey = key
	shop.PluginSecretHash = hashSecret(secret)
	shop.PluginStatus = model.ShopPluginBound
	shop.LastSeenAt = &now
	if err := s.repos.Shop.Save(shop); err != nil {
		return nil, err
	}
	return &dto.PluginBindResult{
		ShopID: shop.ID, ShopName: shop.Name, Platform: shop.Platform,
		PluginKey: key, PluginSecret: secret,
	}, nil
}

func (s *ShopService) AuthenticatePlugin(key, secret string) (*model.MarketplaceShop, error) {
	key = strings.TrimSpace(key)
	secret = strings.TrimSpace(secret)
	if key == "" || secret == "" {
		return nil, ErrPluginAuth
	}
	shop, err := s.repos.Shop.GetByPluginKey(key)
	if err != nil {
		return nil, ErrPluginAuth
	}
	want := shop.PluginSecretHash
	got := hashSecret(secret)
	if subtle.ConstantTimeCompare([]byte(want), []byte(got)) != 1 {
		return nil, ErrPluginAuth
	}
	return shop, nil
}

func (s *ShopService) Heartbeat(shop *model.MarketplaceShop, in *dto.PluginHeartbeatInput) (*dto.PluginHeartbeatResult, error) {
	now := time.Now()
	shop.LastSeenAt = &now
	shop.PluginStatus = model.ShopPluginBound
	if v := strings.TrimSpace(in.PlatformShopID); v != "" {
		shop.PlatformShopID = v
	}
	if v := strings.TrimSpace(in.PlatformShopName); v != "" {
		shop.PlatformShopName = v
	}
	if err := s.repos.Shop.Save(shop); err != nil {
		return nil, err
	}
	interval := s.pluginSyncInterval(shop.TenantID)
	return &dto.PluginHeartbeatResult{
		ShopItem:        s.toItem(shop),
		SyncNow:         pluginShouldSync(shop, now, interval),
		SyncIntervalSec: int(interval.Seconds()),
	}, nil
}

func (s *ShopService) GetPluginSetting() dto.PluginSetting {
	return dto.PluginSetting{PluginSyncIntervalMin: s.repo().PluginSyncMinutes()}
}

func (s *ShopService) SavePluginSetting(in dto.PluginSetting) (dto.PluginSetting, error) {
	item, err := s.repo().SavePluginSyncMinutes(in.PluginSyncIntervalMin)
	if err != nil {
		return dto.PluginSetting{}, err
	}
	return dto.PluginSetting{PluginSyncIntervalMin: item.PluginSyncIntervalMin}, nil
}

func (s *ShopService) pluginSyncInterval(tenantID uint64) time.Duration {
	minutes := s.repos.Shop.ForTenant(tenantID).PluginSyncMinutes()
	return time.Duration(minutes) * time.Minute
}

func (s *ShopService) RequestSync(id uint64) (*dto.ShopItem, error) {
	shop, err := s.repo().Get(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if shop.PluginKey == "" {
		return nil, fmt.Errorf("%w: 店铺尚未绑定插件", ErrBadRequest)
	}
	now := time.Now()
	shop.SyncRequestedAt = &now
	if err := s.repo().Save(shop); err != nil {
		return nil, err
	}
	item := s.toItem(shop)
	return &item, nil
}

func (s *ShopService) Sync(shop *model.MarketplaceShop, in *dto.PluginSyncInput) (*dto.PluginSyncResult, error) {
	now := time.Now()
	shop.LastSeenAt = &now
	shop.LastSyncAt = &now
	shop.SyncRequestedAt = nil
	shop.PluginStatus = model.ShopPluginBound
	if v := strings.TrimSpace(in.PlatformShopID); v != "" {
		shop.PlatformShopID = v
	}
	if v := strings.TrimSpace(in.PlatformShopName); v != "" {
		shop.PlatformShopName = v
	}

	cards := make([]model.AftersaleFilterCard, 0, len(in.Cards))
	for i, c := range in.Cards {
		key := strings.TrimSpace(c.CardKey)
		label := strings.TrimSpace(c.CardLabel)
		group := strings.TrimSpace(c.GroupName)
		if key == "" && label != "" {
			key = group + ":" + label
		}
		if key == "" {
			continue
		}
		sort := c.SortOrder
		if sort == 0 {
			sort = i
		}
		cards = append(cards, model.AftersaleFilterCard{
			TenantID: shop.TenantID, ShopID: shop.ID,
			GroupName: group, CardKey: key, CardLabel: label,
			Count: c.Count, SortOrder: sort, SyncedAt: now,
		})
	}

	tickets := make([]model.AftersaleTicket, 0, len(in.Tickets))
	cardKeys := map[string][]string{}
	for _, t := range in.Tickets {
		aid := strings.TrimSpace(t.PlatformAftersaleID)
		if aid == "" {
			continue
		}
		deadline, action, _ := ParseTimeout(strings.TrimSpace(t.TimeoutText), now)
		tickets = append(tickets, model.AftersaleTicket{
			PlatformAftersaleID: aid,
			OrderNo:             strings.TrimSpace(t.OrderNo),
			ProductTitle:        strings.TrimSpace(t.ProductTitle),
			ProductImage:        strings.TrimSpace(t.ProductImage),
			SKU:                 strings.TrimSpace(t.SKU),
			ProductTags:         strings.TrimSpace(t.ProductTags),
			Tags:                strings.TrimSpace(t.Tags),
			Qty:                 t.Qty,
			BuyQty:              t.BuyQty,
			PayAmount:           strings.TrimSpace(t.PayAmount),
			RefundAmount:        strings.TrimSpace(t.RefundAmount),
			AftersaleType:       strings.TrimSpace(t.AftersaleType),
			Reason:              strings.TrimSpace(t.Reason),
			Status:              strings.TrimSpace(t.Status),
			TimeoutText:         strings.TrimSpace(t.TimeoutText),
			TimeoutAction:       action,
			DeadlineAt:          deadline,
			Dispute:             strings.TrimSpace(t.Dispute),
			Logistics:           strings.TrimSpace(t.Logistics),
			ReturnLogisticsNo:   strings.TrimSpace(t.ReturnLogisticsNo),
			ShipLogisticsNo:     strings.TrimSpace(t.ShipLogisticsNo),
			ApplyTime:           strings.TrimSpace(t.ApplyTime),
			RawJSON:             t.RawJSON,
		})
		cardKeys[aid] = t.CardKeys
	}

	if err := s.repos.Shop.ReplaceCards(shop, cards); err != nil {
		return nil, err
	}
	if err := s.repos.Shop.UpsertTickets(shop, tickets, cardKeys); err != nil {
		return nil, err
	}

	returnCount := 0
	if in.Returns != nil && len(*in.Returns) > 0 {
		returns := make([]model.ReturnPackage, 0, len(*in.Returns))
		for _, item := range *in.Returns {
			aid := strings.TrimSpace(item.PlatformAftersaleID)
			if aid == "" {
				continue
			}
			applyTime := strings.TrimSpace(item.ApplyTime)
			shipTime := strings.TrimSpace(item.ShipTime)
			returnTime, returnedAt, appliedAt := ResolveReturnTimes(
				strings.TrimSpace(item.ReturnTime), applyTime, shipTime, item.TrackJSON,
			)
			returns = append(returns, model.ReturnPackage{
				PlatformAftersaleID: aid,
				OrderNo:             strings.TrimSpace(item.OrderNo),
				ProductTitle:        strings.TrimSpace(item.ProductTitle),
				ProductImage:        strings.TrimSpace(item.ProductImage),
				SKU:                 strings.TrimSpace(item.SKU),
				Qty:                 item.Qty,
				BuyQty:              item.BuyQty,
				PayAmount:           strings.TrimSpace(item.PayAmount),
				RefundAmount:        strings.TrimSpace(item.RefundAmount),
				AftersaleType:       strings.TrimSpace(item.AftersaleType),
				Reason:              strings.TrimSpace(item.Reason),
				Status:              strings.TrimSpace(item.Status),
				OrderInfo:           strings.TrimSpace(item.OrderInfo),
				AftersaleInfo:       strings.TrimSpace(item.AftersaleInfo),
				Logistics:           strings.TrimSpace(item.Logistics),
				LogisticsNo:         strings.TrimSpace(item.LogisticsNo),
				Carrier:             strings.TrimSpace(item.Carrier),
				ReturnLocation:      strings.TrimSpace(item.ReturnLocation),
				ShipTime:            shipTime,
				ApplyTime:           applyTime,
				ReturnTime:          returnTime,
				ReturnedAt:          returnedAt,
				AppliedAt:           appliedAt,
				TrackJSON:           item.TrackJSON,
				RawJSON:             item.RawJSON,
			})
		}
		if err := s.repos.Shop.UpsertReturns(shop, returns); err != nil {
			return nil, err
		}
		returnCount = len(returns)
	}

	shippedCount := 0
	if in.ShippedRefunds != nil && len(*in.ShippedRefunds) > 0 {
		shipped := make([]model.ShippedRefundSuccess, 0, len(*in.ShippedRefunds))
		for _, item := range *in.ShippedRefunds {
			aid := strings.TrimSpace(item.PlatformAftersaleID)
			if aid == "" {
				continue
			}
			logistics := strings.TrimSpace(item.Logistics)
			trackJSON := LimitLogisticsTracksJSON(item.TrackJSON)
			status := ClassifyShippedRefundStatus(logistics, trackJSON, item.LogisticsStatus)
			if status == LogisticsReturned {
				continue
			}
			applyTime := strings.TrimSpace(item.ApplyTime)
			shipped = append(shipped, model.ShippedRefundSuccess{
				PlatformAftersaleID: aid,
				OrderNo:             strings.TrimSpace(item.OrderNo),
				ProductTitle:        strings.TrimSpace(item.ProductTitle),
				ProductImage:        strings.TrimSpace(item.ProductImage),
				SKU:                 strings.TrimSpace(item.SKU),
				ProductTags:         strings.TrimSpace(item.ProductTags),
				Tags:                strings.TrimSpace(item.Tags),
				Qty:                 item.Qty,
				BuyQty:              item.BuyQty,
				PayAmount:           strings.TrimSpace(item.PayAmount),
				RefundAmount:        strings.TrimSpace(item.RefundAmount),
				AftersaleType:       strings.TrimSpace(item.AftersaleType),
				Reason:              strings.TrimSpace(item.Reason),
				Status:              strings.TrimSpace(item.Status),
				OrderInfo:           strings.TrimSpace(item.OrderInfo),
				AftersaleInfo:       strings.TrimSpace(item.AftersaleInfo),
				Logistics:           logistics,
				LogisticsStatus:     status,
				LogisticsNo:         strings.TrimSpace(item.LogisticsNo),
				Carrier:             strings.TrimSpace(item.Carrier),
				ShipTime:            strings.TrimSpace(item.ShipTime),
				TrackJSON:           trackJSON,
				ApplyTime:           applyTime,
				AppliedAt:           ParsePlatformDateTime(applyTime, 0),
				RawJSON:             item.RawJSON,
			})
		}
		if err := s.repos.Shop.UpsertShippedRefunds(shop, shipped); err != nil {
			return nil, err
		}
		shippedCount = len(shipped)
	}

	result := &dto.PluginSyncResult{
		ShopID: shop.ID, CardCount: len(cards), TicketCount: len(tickets),
		ReturnCount: returnCount, ShippedRefundCount: shippedCount, LastSyncAt: formatTime(now),
	}
	if err := s.repos.Shop.Save(shop); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *ShopService) toItem(shop *model.MarketplaceShop) dto.ShopItem {
	status := model.ShopPluginUnbound
	if shop.PluginKey != "" {
		status = model.ShopPluginOffline
		if shop.LastSeenAt != nil && time.Since(*shop.LastSeenAt) <= pluginOnlineSkew {
			status = model.ShopPluginOnline
		}
	}
	item := dto.ShopItem{
		ID: shop.ID, Name: shop.Name, Platform: shop.Platform,
		PlatformLabel: PlatformLabel(shop.Platform),
		BindCode:      shop.BindCode, PluginStatus: status,
		PluginAvailable:  PluginAvailable(shop.Platform),
		PlatformShopID:   shop.PlatformShopID,
		PlatformShopName: shop.PlatformShopName,
		Remark:           shop.Remark,
		CreatedAt:        formatTime(shop.CreatedAt),
		UpdatedAt:        formatTime(shop.UpdatedAt),
	}
	if shop.LastSyncAt != nil {
		item.LastSyncAt = formatTime(*shop.LastSyncAt)
	}
	if shop.LastSeenAt != nil {
		item.LastSeenAt = formatTime(*shop.LastSeenAt)
	}
	item.SyncRequested = shop.SyncRequestedAt != nil
	interval := s.pluginSyncInterval(shop.TenantID)
	item.NextSyncAt = nextSyncHint(shop, interval, time.Now())
	return item
}

func nextSyncHint(shop *model.MarketplaceShop, interval time.Duration, now time.Time) string {
	if shop == nil || shop.PluginKey == "" {
		return ""
	}
	online := shop.LastSeenAt != nil && now.Sub(*shop.LastSeenAt) <= pluginOnlineSkew
	due := pluginShouldSync(shop, now, interval)
	if !online {
		if due || shop.LastSyncAt == nil {
			return "待插件上线后同步"
		}
		return formatTime(shop.LastSyncAt.Add(interval)) + "（插件离线）"
	}
	if due {
		return "下一次心跳"
	}
	return formatTime(shop.LastSyncAt.Add(interval))
}

func pluginShouldSync(shop *model.MarketplaceShop, now time.Time, interval time.Duration) bool {
	if shop == nil {
		return false
	}
	if shop.SyncRequestedAt != nil {
		return true
	}
	if shop.LastSyncAt == nil {
		return true
	}
	return now.Sub(*shop.LastSyncAt) >= interval
}

func toTicketItem(t *model.AftersaleTicket) dto.TicketItem {
	keys := make([]string, 0, len(t.CardKeys))
	for _, c := range t.CardKeys {
		keys = append(keys, c.CardKey)
	}
	item := dto.TicketItem{
		ID: t.ID, PlatformAftersaleID: t.PlatformAftersaleID, OrderNo: t.OrderNo,
		ProductTitle: t.ProductTitle, ProductImage: t.ProductImage, SKU: t.SKU,
		ProductTags: t.ProductTags, Tags: t.Tags,
		Qty: t.Qty, BuyQty: t.BuyQty, PayAmount: t.PayAmount, RefundAmount: t.RefundAmount,
		AftersaleType: t.AftersaleType, Reason: t.Reason, Status: t.Status,
		TimeoutText: t.TimeoutText, TimeoutAction: t.TimeoutAction,
		RemainSeconds:     remainSeconds(t.DeadlineAt, time.Now()),
		Dispute:           t.Dispute,
		Logistics:         t.Logistics,
		ReturnLogisticsNo: t.ReturnLogisticsNo,
		ShipLogisticsNo:   t.ShipLogisticsNo,
		ApplyTime:         t.ApplyTime, CardKeys: keys, SyncedAt: formatTime(t.SyncedAt),
	}
	view := ParseTicketLogistics(t.Logistics)
	item.NeedIntercept = view.Intercept
	if view.HasBuyer {
		item.LogisticsBuyerStatus = view.BuyerStatus
	}
	if view.HasShip {
		item.LogisticsShipStatus = view.ShipStatus
	}
	if t.DeadlineAt != nil {
		item.DeadlineAt = t.DeadlineAt.UTC().Format(time.RFC3339)
	}
	return item
}

func toReturnItem(item *model.ReturnPackage, shopName string) dto.ReturnPackageItem {
	return dto.ReturnPackageItem{
		ID: item.ID, ShopID: item.ShopID, ShopName: shopName,
		PlatformAftersaleID: item.PlatformAftersaleID, OrderNo: item.OrderNo,
		ProductTitle: item.ProductTitle, ProductImage: item.ProductImage, SKU: item.SKU,
		Qty: item.Qty, BuyQty: item.BuyQty, PayAmount: item.PayAmount, RefundAmount: item.RefundAmount,
		AftersaleType: item.AftersaleType, Reason: item.Reason, Status: item.Status,
		OrderInfo: item.OrderInfo, AftersaleInfo: item.AftersaleInfo,
		Logistics: item.Logistics, LogisticsNo: item.LogisticsNo, Carrier: item.Carrier,
		ReturnLocation: item.ReturnLocation, ShipTime: item.ShipTime,
		ApplyTime: item.ApplyTime, ReturnTime: item.ReturnTime,
		SyncedAt: formatTime(item.SyncedAt),
	}
}

func toShippedRefundItem(item *model.ShippedRefundSuccess, shopName string) dto.ShippedRefundItem {
	status := ClassifyShippedRefundStatus(item.Logistics, item.TrackJSON, item.LogisticsStatus)
	tracks := ParseLogisticsTracks(item.TrackJSON)
	outTracks := make([]dto.LogisticsTrack, 0, len(tracks))
	for _, t := range tracks {
		outTracks = append(outTracks, dto.LogisticsTrack{
			Date: t.Date, Title: t.Title, Detail: t.Detail, Text: t.Text,
		})
	}
	return dto.ShippedRefundItem{
		ID: item.ID, ShopID: item.ShopID, ShopName: shopName,
		PlatformAftersaleID: item.PlatformAftersaleID, OrderNo: item.OrderNo,
		ProductTitle: item.ProductTitle, ProductImage: item.ProductImage, SKU: item.SKU,
		ProductTags: item.ProductTags, Tags: item.Tags,
		Qty: item.Qty, BuyQty: item.BuyQty, PayAmount: item.PayAmount, RefundAmount: item.RefundAmount,
		AftersaleType: item.AftersaleType, Reason: item.Reason, Status: item.Status,
		OrderInfo: item.OrderInfo, AftersaleInfo: item.AftersaleInfo,
		Logistics: item.Logistics, LogisticsStatus: status,
		LogisticsNo: item.LogisticsNo, Carrier: item.Carrier, ShipTime: item.ShipTime,
		Tracks: outTracks, Alert: IsLogisticsAlert(status), ApplyTime: item.ApplyTime,
		SyncedAt: formatTime(item.SyncedAt),
	}
}

func hashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

const bindCodeAlphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"

func randomBindCode() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	out := make([]byte, 8)
	for i := range b {
		out[i] = bindCodeAlphabet[int(b[i])%len(bindCodeAlphabet)]
	}
	return string(out), nil
}
