package service

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"aftersalescore/internal/dto"
	"aftersalescore/internal/model"
	"aftersalescore/internal/repo"

	"gorm.io/gorm"
)

const pluginOnlineSkew = 90 * time.Second

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
	out := make([]dto.ShopItem, 0, len(list))
	for i := range list {
		out = append(out, s.toItem(&list[i]))
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

func (s *ShopService) Heartbeat(shop *model.MarketplaceShop, in *dto.PluginHeartbeatInput) (*dto.ShopItem, error) {
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
	item := s.toItem(shop)
	return &item, nil
}

func (s *ShopService) Sync(shop *model.MarketplaceShop, in *dto.PluginSyncInput) (*dto.PluginSyncResult, error) {
	now := time.Now()
	shop.LastSeenAt = &now
	shop.LastSyncAt = &now
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
		tickets = append(tickets, model.AftersaleTicket{
			PlatformAftersaleID: aid,
			OrderNo:             strings.TrimSpace(t.OrderNo),
			ProductTitle:        strings.TrimSpace(t.ProductTitle),
			ProductImage:        strings.TrimSpace(t.ProductImage),
			SKU:                 strings.TrimSpace(t.SKU),
			Qty:                 t.Qty,
			PayAmount:           strings.TrimSpace(t.PayAmount),
			RefundAmount:        strings.TrimSpace(t.RefundAmount),
			AftersaleType:       strings.TrimSpace(t.AftersaleType),
			Reason:              strings.TrimSpace(t.Reason),
			Status:              strings.TrimSpace(t.Status),
			TimeoutText:         strings.TrimSpace(t.TimeoutText),
			Dispute:             strings.TrimSpace(t.Dispute),
			Logistics:           strings.TrimSpace(t.Logistics),
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
	if err := s.repos.Shop.Save(shop); err != nil {
		return nil, err
	}
	return &dto.PluginSyncResult{
		ShopID: shop.ID, CardCount: len(cards), TicketCount: len(tickets),
		LastSyncAt: formatTime(now),
	}, nil
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
	return item
}

func toTicketItem(t *model.AftersaleTicket) dto.TicketItem {
	keys := make([]string, 0, len(t.CardKeys))
	for _, c := range t.CardKeys {
		keys = append(keys, c.CardKey)
	}
	return dto.TicketItem{
		ID: t.ID, PlatformAftersaleID: t.PlatformAftersaleID, OrderNo: t.OrderNo,
		ProductTitle: t.ProductTitle, ProductImage: t.ProductImage, SKU: t.SKU,
		Qty: t.Qty, PayAmount: t.PayAmount, RefundAmount: t.RefundAmount,
		AftersaleType: t.AftersaleType, Reason: t.Reason, Status: t.Status,
		TimeoutText: t.TimeoutText, Dispute: t.Dispute, Logistics: t.Logistics,
		ApplyTime: t.ApplyTime, CardKeys: keys, SyncedAt: formatTime(t.SyncedAt),
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
