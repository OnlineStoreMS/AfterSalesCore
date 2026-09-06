package service

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"aftersalescore/internal/dto"
	"aftersalescore/internal/model"
	"aftersalescore/internal/pkg/pluginsecret"
	"aftersalescore/internal/repo"

	"gorm.io/gorm"
)

const (
	pluginOnlineSkew = 3 * time.Minute
)

var (
	ErrPluginAuth      = errors.New("插件鉴权失败")
	ErrAlreadyBound    = errors.New("店铺已启用 Agent 采集，请先重置")
	ErrBindCodeInvalid = errors.New("绑定码无效")
)

var shopPlatforms = map[string]string{
	model.ShopPlatformDoudian:   "抖店",
	model.ShopPlatformTaobao:    "淘宝",
	model.ShopPlatformPinduoduo: "拼多多",
}

type ShopService struct {
	repos         *repo.Repos
	tenantID      uint64
	codec         *pluginsecret.Codec
	publicBaseURL string
	agents        *AgentsCenterClient
}

func NewShopService(repos *repo.Repos, codec *pluginsecret.Codec, publicBaseURL string, agents *AgentsCenterClient) *ShopService {
	return &ShopService{
		repos:         repos,
		codec:         codec,
		publicBaseURL: strings.TrimRight(strings.TrimSpace(publicBaseURL), "/"),
		agents:        agents,
	}
}

func (s *ShopService) ForTenant(tenantID uint64) *ShopService {
	return &ShopService{
		repos:         s.repos,
		tenantID:      repo.NormalizeTenantID(tenantID),
		codec:         s.codec,
		publicBaseURL: s.publicBaseURL,
		agents:        s.agents,
	}
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

func (s *ShopService) ListOnlineAgentShops(platform string) ([]AgentsOnlineShop, error) {
	if s.agents == nil {
		return nil, fmt.Errorf("AgentsCenter 未配置")
	}
	return s.agents.ListOnlineShops(s.tenantID, platform)
}

func (s *ShopService) CreateFromAgent(in *dto.ShopFromAgentInput) (*dto.ShopItem, error) {
	platform := strings.TrimSpace(in.Platform)
	platformShopID := strings.TrimSpace(in.PlatformShopID)
	if platform == "" || platformShopID == "" {
		return nil, fmt.Errorf("%w: platform/platformShopId 必填", ErrBadRequest)
	}
	if !PluginAvailable(platform) {
		return nil, fmt.Errorf("%w: 该平台暂不支持 Agent 采集", ErrBadRequest)
	}
	jobType := strings.TrimSpace(in.JobType)
	if jobType == "" {
		jobType = "doudian.aftersale"
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = strings.TrimSpace(in.PlatformShopName)
	}
	if name == "" {
		name = platformShopID
	}

	shop, err := s.repo().GetByPlatformShopID(platform, platformShopID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		created, cErr := s.Create(&dto.ShopCreateInput{
			Name:             name,
			Platform:         platform,
			PlatformShopID:   platformShopID,
			PlatformShopName: strings.TrimSpace(in.PlatformShopName),
		})
		if cErr != nil {
			return nil, cErr
		}
		shop, err = s.repo().Get(created.ID)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		if v := strings.TrimSpace(in.PlatformShopName); v != "" {
			shop.PlatformShopName = v
		}
		if name != "" && shop.Name == "" {
			shop.Name = name
		}
		_ = s.repo().Save(shop)
	}

	if shop.PluginKey == "" || shop.PluginSecretEnc == "" {
		if err := s.issuePluginCredentials(shop); err != nil {
			return nil, err
		}
	}

	if jobType == "doudian.aftersale" {
		interval := in.IntervalMinutes
		if interval <= 0 {
			interval = s.repos.Shop.ForTenant(shop.TenantID).PluginSyncMinutes()
		}
		// 新建：立即执行一次，并预约下次 = 现在 + 间隔
		if err := s.upsertAftersaleAssignment(shop, interval, true); err != nil {
			return nil, fmt.Errorf("写入 AgentsCenter 采集任务失败: %w", err)
		}
		s.markCollectTriggered(shop, time.Duration(interval)*time.Minute, time.Now())
		_ = s.repo().Save(shop)
	} else if s.agents != nil {
		if err := s.agents.CreateJob(shop.TenantID, jobType, shop.Platform, shop.PlatformShopID, shop.PlatformShopName, "{}", "aftersales"); err != nil {
			return nil, fmt.Errorf("下发 AgentsCenter 任务失败: %w", err)
		}
	}

	item := s.toItem(shop)
	return &item, nil
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
	platformShopID := strings.TrimSpace(in.PlatformShopID)
	if PluginAvailable(platform) && platformShopID == "" {
		return nil, fmt.Errorf("%w: 请填写平台店铺 ID（需与 WindowsAgent 上报一致）", ErrBadRequest)
	}
	var shop *model.MarketplaceShop
	var lastErr error
	for i := 0; i < 6; i++ {
		code, err := randomBindCode()
		if err != nil {
			return nil, err
		}
		shop = &model.MarketplaceShop{
			Name:             name,
			Platform:         platform,
			BindCode:         code,
			PluginStatus:     model.ShopPluginUnbound,
			PlatformShopID:   platformShopID,
			PlatformShopName: strings.TrimSpace(in.PlatformShopName),
			Remark:           strings.TrimSpace(in.Remark),
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
	if v := strings.TrimSpace(in.PlatformShopID); v != "" {
		shop.PlatformShopID = v
	}
	if v := strings.TrimSpace(in.PlatformShopName); v != "" {
		shop.PlatformShopName = v
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
	shop.PluginSecretEnc = ""
	shop.PluginStatus = model.ShopPluginUnbound
	shop.LastSeenAt = nil
	shop.SyncRequestedAt = nil
	if err := s.repo().Save(shop); err != nil {
		return nil, err
	}
	item := s.toItem(shop)
	return &item, nil
}

func (s *ShopService) EnableAgentCollect(id uint64) (*dto.ShopItem, error) {
	shop, err := s.repo().Get(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if !PluginAvailable(shop.Platform) {
		return nil, fmt.Errorf("%w: 该平台暂不支持 Agent 采集", ErrBadRequest)
	}
	if strings.TrimSpace(shop.PlatformShopID) == "" {
		return nil, fmt.Errorf("%w: 请先填写平台店铺 ID", ErrBadRequest)
	}
	if shop.PluginKey != "" && shop.PluginSecretEnc != "" {
		return nil, ErrAlreadyBound
	}
	if err := s.issuePluginCredentials(shop); err != nil {
		return nil, err
	}
	interval := s.repos.Shop.ForTenant(shop.TenantID).PluginSyncMinutes()
	if err := s.upsertAftersaleAssignment(shop, interval, true); err != nil {
		return nil, fmt.Errorf("写入 AgentsCenter 采集任务失败: %w", err)
	}
	s.markCollectTriggered(shop, time.Duration(interval)*time.Minute, time.Now())
	_ = s.repo().Save(shop)
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

func (s *ShopService) SidebarCounts() (*dto.SidebarCounts, error) {
	out := &dto.SidebarCounts{}
	tabs, err := s.repo().CountServiceTabs(0)
	if err != nil {
		return nil, err
	}
	for _, t := range tabs {
		if t.StatusTab == "待处理" {
			out.PendingServiceOrders = int(t.Count)
			break
		}
	}
	_, interceptTotal, err := s.ListIntercepts(dto.InterceptListQuery{Page: 1, PageSize: 1})
	if err != nil {
		return nil, err
	}
	out.InterceptOrders = int(interceptTotal)
	ticketCounts, err := s.repo().CountPendingTickets()
	if err != nil {
		return nil, err
	}
	for _, n := range ticketCounts {
		out.TicketTotal += n
	}
	openTickets, err := s.repo().ListOpenTickets(0)
	if err != nil {
		return nil, err
	}
	for i := range openTickets {
		if MatchShopTicketKind(&openTickets[i], dto.TicketKindBuyerReturnPickup) {
			out.BuyerReturnPickup++
		}
		if MatchShopTicketKind(&openTickets[i], dto.TicketKindReviewShippedRefund) {
			out.ReviewShippedRefund++
		}
		if MatchShopTicketKind(&openTickets[i], dto.TicketKindBuyerReturnSigned) {
			out.BuyerReturnSigned++
		}
	}
	return out, nil
}

func MatchShopTicketKind(t *model.AftersaleTicket, kind string) bool {
	if t == nil {
		return false
	}
	switch kind {
	case dto.TicketKindBuyerReturnPickup:
		if !ticketHasGroup(t, "待商家收/发货") {
			return false
		}
		view := ParseTicketLogistics(t.Logistics)
		return view.HasBuyer && view.BuyerStatus == LogisticsAwaitPickup
	case dto.TicketKindReviewShippedRefund:
		return ticketHasCard(t, "待商家审核", "已发货退款")
	case dto.TicketKindBuyerReturnSigned:
		if !ticketHasCard(t, "待商家收/发货", "全部待收货/发货") {
			return false
		}
		view := ParseTicketLogistics(t.Logistics)
		return view.HasBuyer && view.BuyerStatus == LogisticsSigned
	default:
		return false
	}
}

func ticketHasGroup(t *model.AftersaleTicket, group string) bool {
	for _, c := range t.CardKeys {
		if strings.HasPrefix(c.CardKey, group+":") || strings.Contains(c.CardKey, group) {
			return true
		}
	}
	return false
}

func ticketHasCard(t *model.AftersaleTicket, group, label string) bool {
	want := group + ":" + label
	for _, c := range t.CardKeys {
		if c.CardKey == want {
			return true
		}
		if strings.Contains(c.CardKey, group) && strings.Contains(c.CardKey, label) {
			return true
		}
	}
	return false
}

func (s *ShopService) ListShopTickets(q dto.ShopTicketListQuery) ([]dto.TicketItem, int64, error) {
	if q.Kind != dto.TicketKindBuyerReturnPickup && q.Kind != dto.TicketKindReviewShippedRefund && q.Kind != dto.TicketKindBuyerReturnSigned {
		return nil, 0, fmt.Errorf("%w: 无效筛选", ErrBadRequest)
	}
	if q.ShopID > 0 {
		if _, err := s.repo().Get(q.ShopID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, 0, ErrNotFound
			}
			return nil, 0, err
		}
	}
	list, err := s.repo().ListOpenTickets(q.ShopID)
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
	kw := strings.ToLower(strings.TrimSpace(q.Keyword))
	filtered := make([]model.AftersaleTicket, 0, len(list))
	for i := range list {
		t := &list[i]
		if !MatchShopTicketKind(t, q.Kind) {
			continue
		}
		if kw != "" && !shopTicketKeywordMatch(t, names[t.ShopID], kw) {
			continue
		}
		filtered = append(filtered, *t)
	}
	total := int64(len(filtered))
	page, pageSize := q.Page, q.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	start := (page - 1) * pageSize
	if start >= len(filtered) {
		return []dto.TicketItem{}, total, nil
	}
	end := start + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	out := make([]dto.TicketItem, 0, end-start)
	for i := start; i < end; i++ {
		item := toTicketItem(&filtered[i])
		item.ShopName = names[filtered[i].ShopID]
		out = append(out, item)
	}
	return out, total, nil
}

func shopTicketKeywordMatch(t *model.AftersaleTicket, shopName, kw string) bool {
	blob := strings.ToLower(strings.Join([]string{
		shopName, t.PlatformAftersaleID, t.OrderNo, t.ProductTitle, t.SKU, t.Status, t.Logistics,
		t.ReturnLogisticsNo, t.ShipLogisticsNo, t.AftersaleType,
	}, " "))
	return strings.Contains(blob, kw)
}

func (s *ShopService) ListServiceOrders(q dto.ServiceOrderListQuery) ([]dto.ServiceOrderItem, []dto.ServiceTabCount, int64, error) {
	if q.ShopID > 0 {
		if _, err := s.repo().Get(q.ShopID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, nil, 0, ErrNotFound
			}
			return nil, nil, 0, err
		}
	}
	list, total, err := s.repo().ListServiceOrders(repo.ServiceOrderListFilter{
		ShopID: q.ShopID, StatusTab: q.StatusTab, Keyword: q.Keyword, Page: q.Page, PageSize: q.PageSize,
	})
	if err != nil {
		return nil, nil, 0, err
	}
	tabs, err := s.repo().CountServiceTabs(q.ShopID)
	if err != nil {
		return nil, nil, 0, err
	}
	shops, err := s.repo().List()
	if err != nil {
		return nil, nil, 0, err
	}
	names := make(map[uint64]string, len(shops))
	for i := range shops {
		names[shops[i].ID] = shops[i].Name
	}
	counts := make([]dto.ServiceTabCount, 0, len(tabs))
	for _, t := range tabs {
		counts = append(counts, dto.ServiceTabCount{StatusTab: t.StatusTab, Count: t.Count})
	}
	out := make([]dto.ServiceOrderItem, 0, len(list))
	for i := range list {
		out = append(out, toServiceOrderItem(&list[i], names[list[i].ShopID]))
	}
	return out, counts, total, nil
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
		ReturnLogisticsNo: t.ReturnLogisticsNo, Tracks: toDTOTracks(t.TrackJSON),
		ApplyTime: t.ApplyTime, SyncedAt: formatTime(t.SyncedAt),
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
		LogisticsNo: p.LogisticsNo, Carrier: p.Carrier, Tracks: toDTOTracks(p.TrackJSON),
		ApplyTime: p.ApplyTime, SyncedAt: formatTime(p.SyncedAt),
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

func (s *ShopService) ListReturnRefunds(q dto.ReturnRefundListQuery) ([]dto.ReturnRefundItem, int64, error) {
	if q.ShopID > 0 {
		if _, err := s.repo().Get(q.ShopID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, 0, ErrNotFound
			}
			return nil, 0, err
		}
	}
	list, _, err := s.repo().ListReturnRefunds(repo.ReturnRefundListFilter{
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
	out := make([]dto.ReturnRefundItem, 0, len(list))
	for i := range list {
		item := toReturnRefundItem(&list[i], names[list[i].ShopID])
		if wantStatus != "" && item.LogisticsStatus != wantStatus {
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
		return []dto.ReturnRefundItem{}, total, nil
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
	if err := s.ForTenant(shop.TenantID).issuePluginCredentials(shop); err != nil {
		return nil, err
	}
	secret, err := s.codec.Decrypt(shop.PluginSecretEnc)
	if err != nil {
		return nil, err
	}
	return &dto.PluginBindResult{
		ShopID: shop.ID, ShopName: shop.Name, Platform: shop.Platform,
		PluginKey: shop.PluginKey, PluginSecret: secret,
	}, nil
}

func (s *ShopService) issuePluginCredentials(shop *model.MarketplaceShop) error {
	key, err := randomHex(16)
	if err != nil {
		return err
	}
	secret, err := randomHex(24)
	if err != nil {
		return err
	}
	if s.codec == nil {
		return fmt.Errorf("plugin secret codec 未初始化")
	}
	enc, err := s.codec.Encrypt(secret)
	if err != nil {
		return err
	}
	now := time.Now()
	shop.PluginKey = key
	shop.PluginSecretHash = hashSecret(secret)
	shop.PluginSecretEnc = enc
	shop.PluginStatus = model.ShopPluginBound
	shop.LastSeenAt = &now
	return s.repo().Save(shop)
}

func (s *ShopService) AgentCredentialByPlatformShop(tenantID uint64, platform, platformShopID string) (*dto.AgentShopCredential, error) {
	platform = strings.TrimSpace(platform)
	platformShopID = strings.TrimSpace(platformShopID)
	if platform == "" || platformShopID == "" {
		return nil, fmt.Errorf("%w: platform/platformShopId 必填", ErrBadRequest)
	}
	shop, err := s.repos.Shop.ForTenant(tenantID).GetByPlatformShopID(platform, platformShopID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.credentialFromShop(shop)
}

func (s *ShopService) credentialFromShop(shop *model.MarketplaceShop) (*dto.AgentShopCredential, error) {
	if shop.PluginKey == "" || shop.PluginSecretEnc == "" {
		return nil, fmt.Errorf("%w: 店铺尚未启用 Agent 采集", ErrBadRequest)
	}
	if s.codec == nil {
		return nil, fmt.Errorf("plugin secret codec 未初始化")
	}
	secret, err := s.codec.Decrypt(shop.PluginSecretEnc)
	if err != nil {
		return nil, fmt.Errorf("解密采集凭证失败: %w", err)
	}
	return &dto.AgentShopCredential{
		TenantID:         shop.TenantID,
		ShopID:           shop.ID,
		ShopName:         shop.Name,
		Platform:         shop.Platform,
		PlatformShopID:   shop.PlatformShopID,
		PlatformShopName: shop.PlatformShopName,
		PluginKey:        shop.PluginKey,
		PluginSecret:     secret,
		APIBase:          s.publicBaseURL,
	}, nil
}

func (s *ShopService) buildAftersaleParamsJSON(shop *model.MarketplaceShop) (string, error) {
	cred, err := s.credentialFromShop(shop)
	if err != nil {
		return "", err
	}
	payload := map[string]any{
		"apiBase":          cred.APIBase,
		"shopId":           cred.ShopID,
		"shopName":         cred.ShopName,
		"platform":         cred.Platform,
		"pluginKey":        cred.PluginKey,
		"pluginSecret":     cred.PluginSecret,
		"platformShopId":   cred.PlatformShopID,
		"platformShopName": cred.PlatformShopName,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (s *ShopService) dispatchAftersaleJob(shop *model.MarketplaceShop) error {
	interval := s.repos.Shop.ForTenant(shop.TenantID).PluginSyncMinutes()
	return s.upsertAftersaleAssignment(shop, interval, true)
}

func (s *ShopService) upsertAftersaleAssignment(shop *model.MarketplaceShop, intervalMinutes int, triggerNow bool) error {
	if s.agents == nil {
		return fmt.Errorf("AgentsCenter 未配置")
	}
	if strings.TrimSpace(shop.PlatformShopID) == "" {
		return fmt.Errorf("%w: 平台店铺 ID 为空", ErrBadRequest)
	}
	params, err := s.buildAftersaleParamsJSON(shop)
	if err != nil {
		return err
	}
	return s.agents.UpsertAftersaleAssignment(
		shop.TenantID,
		shop.Platform,
		shop.PlatformShopID,
		shop.PlatformShopName,
		params,
		intervalMinutes,
		triggerNow,
	)
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
	if err := s.repos.Shop.TouchHeartbeat(shop); err != nil {
		return nil, err
	}
	interval := s.pluginSyncInterval(shop.TenantID)
	return &dto.PluginHeartbeatResult{
		ShopItem:        s.toItem(shop),
		SyncNow:         agentCollectDue(shop, now),
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
	minutes := item.PluginSyncIntervalMin
	if err := s.syncAgentAssignmentsInterval(minutes); err != nil {
		return dto.PluginSetting{PluginSyncIntervalMin: minutes}, fmt.Errorf("间隔已保存，但同步 Agents 任务失败: %w", err)
	}
	return dto.PluginSetting{PluginSyncIntervalMin: minutes}, nil
}

// syncAgentAssignmentsInterval 将本租户已启用采集的店铺订阅间隔/参数同步到 Agents 中心。
func (s *ShopService) syncAgentAssignmentsInterval(intervalMinutes int) error {
	if s.agents == nil {
		return nil
	}
	list, err := s.repo().List()
	if err != nil {
		return err
	}
	now := time.Now()
	interval := time.Duration(intervalMinutes) * time.Minute
	var firstErr error
	for i := range list {
		shop := &list[i]
		if shop.PluginKey == "" || shop.PluginSecretEnc == "" || strings.TrimSpace(shop.PlatformShopID) == "" {
			continue
		}
		// 按新间隔重算下次执行：从最近一次触发/同步起算，过期则改为 now+间隔
		base := now
		if shop.SyncRequestedAt != nil {
			base = *shop.SyncRequestedAt
		} else if shop.LastSyncAt != nil {
			base = *shop.LastSyncAt
		}
		next := base.Add(interval)
		if !next.After(now) {
			next = now.Add(interval)
		}
		shop.AgentNextRunAt = &next
		_ = s.repos.Shop.ForTenant(shop.TenantID).Save(shop)

		if err := s.upsertAftersaleAssignment(shop, intervalMinutes, false); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
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
	if shop.PluginKey == "" || shop.PluginSecretEnc == "" {
		return nil, fmt.Errorf("%w: 店铺尚未启用 Agent 采集", ErrBadRequest)
	}
	if strings.TrimSpace(shop.PlatformShopID) == "" {
		return nil, fmt.Errorf("%w: 请先填写平台店铺 ID", ErrBadRequest)
	}
	now := time.Now()
	if err := s.dispatchAftersaleJob(shop); err != nil {
		return nil, fmt.Errorf("触发 AgentsCenter 采集失败: %w", err)
	}
	interval := s.pluginSyncInterval(shop.TenantID)
	s.markCollectTriggered(shop, interval, now)
	if err := s.repo().Save(shop); err != nil {
		return nil, err
	}
	item := s.toItem(shop)
	return &item, nil
}

// DispatchDueAgentJobs 按 AgentNextRunAt 到期触发已有采集任务再执行。
func (s *ShopService) DispatchDueAgentJobs() (int, error) {
	list, err := s.repos.Shop.ListBoundAgentCollectable()
	if err != nil {
		return 0, err
	}
	now := time.Now()
	n := 0
	for i := range list {
		shop := &list[i]
		interval := s.pluginSyncInterval(shop.TenantID)
		s.ensureAgentNextRunAt(shop, interval, now)
		if !agentCollectDue(shop, now) {
			continue
		}
		if err := s.dispatchAftersaleJob(shop); err != nil {
			continue
		}
		s.markCollectTriggered(shop, interval, now)
		_ = s.repos.Shop.ForTenant(shop.TenantID).Save(shop)
		n++
	}
	return n, nil
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
			TrackJSON:           LimitLogisticsTracksJSON(t.TrackJSON),
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
				TrackJSON:           LimitLogisticsTracksJSON(item.TrackJSON),
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

	returnRefundCount := 0
	if in.ReturnRefunds != nil {
		refunds := make([]model.ReturnRefundSuccess, 0, len(*in.ReturnRefunds))
		for _, item := range *in.ReturnRefunds {
			aid := strings.TrimSpace(item.PlatformAftersaleID)
			if aid == "" {
				continue
			}
			logistics := strings.TrimSpace(item.Logistics)
			trackJSON := LimitLogisticsTracksJSON(item.TrackJSON)
			status := ClassifyLogisticsWithTracks(logistics, trackJSON)
			if status == "" {
				status = strings.TrimSpace(item.LogisticsStatus)
			}
			applyTime := strings.TrimSpace(item.ApplyTime)
			refunds = append(refunds, model.ReturnRefundSuccess{
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
				AftersaleType:       firstNonEmpty(strings.TrimSpace(item.AftersaleType), "退货退款"),
				Reason:              strings.TrimSpace(item.Reason),
				Status:              firstNonEmpty(strings.TrimSpace(item.Status), "退款成功"),
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
		if err := s.repos.Shop.UpsertReturnRefunds(shop, refunds); err != nil {
			return nil, err
		}
		returnRefundCount = len(refunds)
	}

	serviceCount := 0
	if in.ServiceOrders != nil {
		orders := make([]model.ServiceOrder, 0, len(*in.ServiceOrders))
		for _, o := range *in.ServiceOrders {
			sid := strings.TrimSpace(o.PlatformServiceID)
			if sid == "" {
				continue
			}
			deadline, action := deadlineFromUnix(o.DelayEndTime, strings.TrimSpace(o.TimeoutText), now)
			raw := o.RawJSON
			if strings.TrimSpace(raw) == "" {
				raw = "{}"
			}
			orders = append(orders, model.ServiceOrder{
				PlatformServiceID: sid,
				OrderNo:           strings.TrimSpace(o.OrderNo),
				ProductTitle:      strings.TrimSpace(o.ProductTitle),
				ProductImage:      strings.TrimSpace(o.ProductImage),
				ProductContent:    strings.TrimSpace(o.ProductContent),
				BuyerNick:         strings.TrimSpace(o.BuyerNick),
				CreateSource:      strings.TrimSpace(o.CreateSource),
				BusinessType:      strings.TrimSpace(o.BusinessType),
				OrderType:         strings.TrimSpace(o.OrderType),
				Tags:              strings.TrimSpace(o.Tags),
				StatusTab:         strings.TrimSpace(o.StatusTab),
				Status:            strings.TrimSpace(o.Status),
				TimeoutText:       strings.TrimSpace(o.TimeoutText),
				TimeoutAction:     action,
				DeadlineAt:        deadline,
				DelayEndTime:      o.DelayEndTime,
				Detail:            strings.TrimSpace(o.Detail),
				Solution:          strings.TrimSpace(o.Solution),
				LastLog:           strings.TrimSpace(o.LastLog),
				LastLogTime:       strings.TrimSpace(o.LastLogTime),
				CreateTime:        strings.TrimSpace(o.CreateTime),
				RawJSON:           raw,
			})
		}
		if err := s.repos.Shop.UpsertServiceOrders(shop, orders); err != nil {
			return nil, err
		}
		serviceCount = len(orders)
	}

	result := &dto.PluginSyncResult{
		ShopID: shop.ID, CardCount: len(cards), TicketCount: len(tickets),
		ReturnCount: returnCount, ShippedRefundCount: shippedCount, ReturnRefundCount: returnRefundCount,
		ServiceOrderCount: serviceCount,
		LastSyncAt: formatTime(now),
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
	if shop.AgentNextRunAt != nil {
		if !shop.AgentNextRunAt.After(now) {
			if shop.SyncRequestedAt != nil {
				return "已请求，等待 Agent 执行"
			}
			return "待自动执行"
		}
		return formatTime(*shop.AgentNextRunAt)
	}
	if shop.LastSyncAt != nil {
		return formatTime(shop.LastSyncAt.Add(interval))
	}
	return "创建后将按间隔执行"
}

// agentCollectDue 新框架：仅当预约的下次执行时间已到。
func agentCollectDue(shop *model.MarketplaceShop, now time.Time) bool {
	if shop == nil || shop.AgentNextRunAt == nil {
		return false
	}
	return !shop.AgentNextRunAt.After(now)
}

// markCollectTriggered 记录本次触发，并预约下次执行。
func (s *ShopService) markCollectTriggered(shop *model.MarketplaceShop, interval time.Duration, now time.Time) {
	if shop == nil {
		return
	}
	if interval <= 0 {
		interval = 30 * time.Minute
	}
	shop.SyncRequestedAt = &now
	next := now.Add(interval)
	shop.AgentNextRunAt = &next
}

// ensureAgentNextRunAt 给老数据补下次执行时间，避免再按「从未同步」狂触发。
func (s *ShopService) ensureAgentNextRunAt(shop *model.MarketplaceShop, interval time.Duration, now time.Time) {
	if shop == nil || shop.AgentNextRunAt != nil {
		return
	}
	if interval <= 0 {
		interval = 30 * time.Minute
	}
	var next time.Time
	switch {
	case shop.LastSyncAt != nil:
		next = shop.LastSyncAt.Add(interval)
	case shop.SyncRequestedAt != nil:
		next = shop.SyncRequestedAt.Add(interval)
	default:
		next = now.Add(interval)
	}
	if !next.After(now) {
		next = now.Add(interval)
	}
	shop.AgentNextRunAt = &next
	_ = s.repos.Shop.ForTenant(shop.TenantID).Save(shop)
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
		Tracks:            toDTOTracks(t.TrackJSON),
		ShopID:            t.ShopID,
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

func toServiceOrderItem(o *model.ServiceOrder, shopName string) dto.ServiceOrderItem {
	item := dto.ServiceOrderItem{
		ID: o.ID, ShopID: o.ShopID, ShopName: shopName,
		PlatformServiceID: o.PlatformServiceID, OrderNo: o.OrderNo,
		ProductTitle: o.ProductTitle, ProductImage: o.ProductImage, ProductContent: o.ProductContent,
		BuyerNick: o.BuyerNick, CreateSource: o.CreateSource,
		BusinessType: o.BusinessType, OrderType: o.OrderType, Tags: o.Tags,
		StatusTab: o.StatusTab, Status: o.Status,
		TimeoutText: o.TimeoutText, TimeoutAction: o.TimeoutAction,
		RemainSeconds: remainSeconds(o.DeadlineAt, time.Now()),
		Detail:        o.Detail, Solution: o.Solution,
		LastLog: o.LastLog, LastLogTime: o.LastLogTime, CreateTime: o.CreateTime,
		SyncedAt: formatTime(o.SyncedAt),
	}
	if o.DeadlineAt != nil {
		item.DeadlineAt = o.DeadlineAt.UTC().Format(time.RFC3339)
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
		Tracks:   toDTOTracks(item.TrackJSON),
		SyncedAt: formatTime(item.SyncedAt),
	}
}

func toReturnRefundItem(item *model.ReturnRefundSuccess, shopName string) dto.ReturnRefundItem {
	status := ClassifyLogisticsWithTracks(item.Logistics, item.TrackJSON)
	if status == "" {
		status = item.LogisticsStatus
	}
	return dto.ReturnRefundItem{
		ID: item.ID, ShopID: item.ShopID, ShopName: shopName,
		PlatformAftersaleID: item.PlatformAftersaleID, OrderNo: item.OrderNo,
		ProductTitle: item.ProductTitle, ProductImage: item.ProductImage, SKU: item.SKU,
		ProductTags: item.ProductTags, Tags: item.Tags,
		Qty: item.Qty, BuyQty: item.BuyQty, PayAmount: item.PayAmount, RefundAmount: item.RefundAmount,
		AftersaleType: firstNonEmpty(item.AftersaleType, "退货退款"), Reason: item.Reason,
		Status: firstNonEmpty(item.Status, "退款成功"),
		OrderInfo: item.OrderInfo, AftersaleInfo: item.AftersaleInfo,
		Logistics: item.Logistics, LogisticsStatus: status,
		LogisticsNo: item.LogisticsNo, Carrier: item.Carrier, ShipTime: item.ShipTime,
		Tracks: toDTOTracks(item.TrackJSON), ApplyTime: item.ApplyTime,
		SyncedAt: formatTime(item.SyncedAt),
	}
}

func toDTOTracks(raw string) []dto.LogisticsTrack {
	tracks := ParseLogisticsTracks(raw)
	if len(tracks) == 0 {
		return nil
	}
	out := make([]dto.LogisticsTrack, 0, len(tracks))
	for _, t := range tracks {
		out = append(out, dto.LogisticsTrack{
			Date: t.Date, Title: t.Title, Detail: t.Detail, Text: t.Text,
		})
	}
	return out
}

func toShippedRefundItem(item *model.ShippedRefundSuccess, shopName string) dto.ShippedRefundItem {
	status := ClassifyShippedRefundStatus(item.Logistics, item.TrackJSON, item.LogisticsStatus)
	outTracks := toDTOTracks(item.TrackJSON)
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
