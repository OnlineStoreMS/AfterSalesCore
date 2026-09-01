package service

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"aftersalescore/internal/dto"
	"aftersalescore/internal/feishu"
	"aftersalescore/internal/model"
	"aftersalescore/internal/repo"
)

type NotificationService struct {
	repos    *repo.Repos
	tenantID uint64
	feishu   *feishu.Client
}

func NewNotificationService(repos *repo.Repos) *NotificationService {
	return &NotificationService{repos: repos, feishu: feishu.NewClient()}
}

func (s *NotificationService) ForTenant(tenantID uint64) *NotificationService {
	return &NotificationService{repos: s.repos, tenantID: repo.NormalizeTenantID(tenantID), feishu: s.feishu}
}

func (s *NotificationService) shopRepo() *repo.ShopRepo {
	return s.repos.Shop.ForTenant(s.tenantID)
}

func (s *NotificationService) GetView() (*dto.NotificationView, error) {
	data, err := s.repos.Notification.Load(s.tenantID)
	if err != nil {
		return nil, err
	}
	if err := s.dropLegacyServiceNotification(data); err != nil {
		return nil, err
	}
	return s.buildView(data)
}

func (s *NotificationService) dropLegacyServiceNotification(data dto.NotificationData) error {
	cards, err := s.shopRepo().ListCardsForTenant()
	if err != nil {
		return err
	}
	cleaned := sanitizeScenarios(data.Config.Scenarios, cards)
	if !stringSlicesEqual(data.Config.Scenarios, cleaned) {
		data.Config.Scenarios = cleaned
		if _, err := s.repos.Notification.SaveConfig(s.tenantID, data.Config); err != nil {
			return err
		}
	}
	if pruneServiceNotified(data.State.Notified) {
		return s.repos.Notification.UpdateState(s.tenantID, func(st *dto.NotificationState) error {
			if st.Notified == nil {
				return nil
			}
			pruneServiceNotified(st.Notified)
			return nil
		})
	}
	return nil
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func pruneServiceNotified(notified map[string]string) bool {
	if len(notified) == 0 {
		return false
	}
	changed := false
	for key := range notified {
		if isLegacyServiceNotificationKey(key) {
			delete(notified, key)
			changed = true
		}
	}
	return changed
}

func isLegacyServiceNotificationKey(key string) bool {
	parts := strings.Split(key, ":")
	if len(parts) >= 2 && parts[1] == "service" {
		return true
	}
	return strings.Contains(key, ":service:")
}

func (s *NotificationService) ResetState() (*dto.NotificationView, int, error) {
	cleared, err := s.repos.Notification.ResetState(s.tenantID)
	if err != nil {
		return nil, 0, err
	}
	view, err := s.GetView()
	return view, cleared, err
}

func (s *NotificationService) SaveConfig(in dto.NotificationConfig) (*dto.NotificationView, error) {
	cards, err := s.shopRepo().ListCardsForTenant()
	if err != nil {
		return nil, err
	}
	in.Scenarios = sanitizeScenarios(in.Scenarios, cards)
	ids, err := s.resolveShopIDs(in.ShopIDs)
	if err != nil {
		return nil, err
	}
	if in.Enabled && len(ids) == 0 {
		return nil, fmt.Errorf("%w: 请先在「店铺管理」添加店铺", ErrBadRequest)
	}
	data, err := s.repos.Notification.SaveConfig(s.tenantID, in)
	if err != nil {
		return nil, err
	}
	return s.buildView(data)
}

func (s *NotificationService) buildView(data dto.NotificationData) (*dto.NotificationView, error) {
	shops, err := s.shopRepo().List()
	if err != nil {
		return nil, err
	}
	cards, err := s.shopRepo().ListCardsForTenant()
	if err != nil {
		return nil, err
	}
	shopOpts := make([]dto.NotificationShopOption, 0, len(shops))
	for _, shop := range shops {
		shopOpts = append(shopOpts, dto.NotificationShopOption{
			ID:   strconv.FormatUint(shop.ID, 10),
			Name: shopDisplayName(&shop),
		})
	}
	data.Config.Scenarios = sanitizeScenarios(data.Config.Scenarios, cards)
	state := data.State
	state.Notified = nil
	return &dto.NotificationView{
		Config:    toNotificationConfigView(data.Config),
		State:     state,
		Scenarios: scenarioOptions(cards),
		Shops:     shopOpts,
	}, nil
}

func toNotificationConfigView(cfg dto.NotificationConfig) dto.NotificationConfigView {
	return dto.NotificationConfigView{
		Enabled:             cfg.Enabled,
		WebhookURL:          cfg.WebhookURL,
		PollIntervalMinutes: cfg.PollIntervalMinutes,
		Scenarios:           append([]string(nil), cfg.Scenarios...),
		ShopIDs:             append([]string(nil), cfg.ShopIDs...),
		SecretSet:           cfg.Secret != "",
		AppID:               cfg.AppID,
		AppSecretSet:        cfg.AppSecret != "",
	}
}

func (s *NotificationService) resolveShopIDs(selected []string) ([]uint64, error) {
	all, err := s.shopRepo().List()
	if err != nil {
		return nil, err
	}
	known := map[uint64]struct{}{}
	allIDs := make([]uint64, 0, len(all))
	for _, shop := range all {
		known[shop.ID] = struct{}{}
		allIDs = append(allIDs, shop.ID)
	}
	if len(selected) == 0 {
		return allIDs, nil
	}
	ids := make([]uint64, 0, len(selected))
	for _, raw := range selected {
		id, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
		if err != nil || id == 0 {
			return nil, fmt.Errorf("%w: 无效店铺 %s", ErrBadRequest, raw)
		}
		if _, ok := known[id]; !ok {
			return nil, fmt.Errorf("%w: 店铺不存在 %d", ErrBadRequest, id)
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("%w: 至少选择一个通知店铺", ErrBadRequest)
	}
	return ids, nil
}

func (s *NotificationService) Test(ctx context.Context, text string) error {
	data, err := s.repos.Notification.Load(s.tenantID)
	if err != nil {
		return err
	}
	cfg := data.Config
	if cfg.WebhookURL == "" {
		return fmt.Errorf("请先配置 Webhook 地址")
	}
	if text == "" {
		text = "这是一条测试消息"
	}
	card := feishu.InteractiveCard{
		Title:    "售后管理 · 测试通知",
		Template: "blue",
		Markdown: fmt.Sprintf("**说明：** %s\n\n<font color='grey'>若能看到本条彩色卡片，说明 Webhook 配置正确。</font>", escapeLarkMD(text)),
	}
	return s.feishu.SendInteractiveCard(ctx, cfg.WebhookURL, cfg.Secret, card)
}

func (s *NotificationService) TestBarcode(ctx context.Context) error {
	data, err := s.repos.Notification.Load(s.tenantID)
	if err != nil {
		return err
	}
	cfg := data.Config
	if cfg.WebhookURL == "" {
		return fmt.Errorf("请先配置 Webhook 地址")
	}
	if cfg.AppID == "" || cfg.AppSecret == "" {
		return fmt.Errorf("请先配置飞书应用 ID 和 App Secret")
	}
	const sampleSid = "YT1234567890"
	card := feishu.InteractiveCard{
		Title:    "售后管理 · 条形码测试",
		Template: "blue",
		Markdown: fmt.Sprintf("**退货物流：** %s\n\n<font color='grey'>若下方出现条形码，说明飞书应用配置正确。</font>", escapeLarkMD(sampleSid)),
	}
	if err := s.attachLogisticsBarcode(ctx, cfg, &card, sampleSid); err != nil {
		return fmt.Errorf("条形码上传失败: %w", err)
	}
	if card.FooterImgKey == "" {
		return fmt.Errorf("条形码上传失败: 未获得 image_key")
	}
	return s.feishu.SendInteractiveCard(ctx, cfg.WebhookURL, cfg.Secret, card)
}

func (s *NotificationService) Enabled() bool {
	data, err := s.repos.Notification.Load(s.tenantID)
	if err != nil {
		return false
	}
	return data.Config.Enabled && data.Config.WebhookURL != ""
}

func (s *NotificationService) PollInterval() time.Duration {
	data, err := s.repos.Notification.Load(s.tenantID)
	if err != nil || !data.Config.Enabled {
		return 5 * time.Minute
	}
	mins := data.Config.PollIntervalMinutes
	if mins < 5 {
		mins = 5
	}
	return time.Duration(mins) * time.Minute
}

func (s *NotificationService) NotifyShop(ctx context.Context, tenantID, shopID uint64) {
	if tenantID == 0 || shopID == 0 {
		return
	}
	svc := s.ForTenant(tenantID)
	if !svc.Enabled() {
		return
	}
	if _, err := svc.RunPoll(ctx, shopID); err != nil {
		log.Printf("[notification] shop %d: %v", shopID, err)
	}
}

func (s *NotificationService) RunPoll(ctx context.Context, onlyShopID uint64) (*dto.NotificationRunResult, error) {
	data, err := s.repos.Notification.Load(s.tenantID)
	if err != nil {
		return nil, err
	}
	cfg := data.Config
	result := &dto.NotificationRunResult{}
	now := time.Now()
	runAt := now.Format("2006-01-02 15:04:05")

	updateState := func(ok bool, sent int, errMsg, barcodeErr string) {
		_ = s.repos.Notification.UpdateState(s.tenantID, func(st *dto.NotificationState) error {
			st.LastRunAt = runAt
			st.LastRunOK = ok
			st.LastError = errMsg
			st.LastSentCount = sent
			st.LastBarcodeError = barcodeErr
			return nil
		})
	}

	if !cfg.Enabled {
		updateState(true, 0, "", "")
		return result, nil
	}
	if cfg.WebhookURL == "" {
		err := fmt.Errorf("webhook url is empty")
		updateState(false, 0, err.Error(), "")
		return nil, err
	}
	if len(cfg.Scenarios) == 0 {
		updateState(true, 0, "", "")
		return result, nil
	}

	shopIDs, err := s.resolveShopIDs(cfg.ShopIDs)
	if err != nil {
		updateState(false, 0, err.Error(), "")
		return nil, err
	}
	if onlyShopID != 0 {
		filtered := shopIDs[:0]
		for _, id := range shopIDs {
			if id == onlyShopID {
				filtered = append(filtered, id)
			}
		}
		shopIDs = filtered
		if len(shopIDs) == 0 {
			return result, nil
		}
	}
	if len(shopIDs) == 0 {
		err := fmt.Errorf("当前租户无可用店铺，请先在「店铺管理」添加")
		updateState(false, 0, err.Error(), "")
		return nil, err
	}

	cards, err := s.shopRepo().ListCardsForTenant()
	if err != nil {
		updateState(false, 0, err.Error(), "")
		return nil, err
	}

	sent := 0
	skipped := 0
	barcodeWarnings := 0
	var lastBarcodeError string
	var sendErr error
	notified := data.State.Notified
	if notified == nil {
		notified = map[string]string{}
	}

	for _, shopID := range shopIDs {
		shop, err := s.shopRepo().Get(shopID)
		if err != nil {
			sendErr = fmt.Errorf("店铺 %d: %w", shopID, err)
			break
		}
		shopName := shopDisplayName(shop)
		for _, scenario := range cfg.Scenarios {
			if !scenarioAllowed(scenario, cards) {
				continue
			}
			nSent, nSkip, nBar, barErr, err := s.sendScenario(ctx, cfg, shop, shopName, scenario, cards, notified, now)
			sent += nSent
			skipped += nSkip
			barcodeWarnings += nBar
			if barErr != "" {
				lastBarcodeError = barErr
			}
			if err != nil {
				sendErr = err
				break
			}
		}
		if sendErr != nil {
			break
		}
	}

	result.Sent = sent
	result.Skipped = skipped
	result.BarcodeWarnings = barcodeWarnings
	result.LastBarcodeError = lastBarcodeError
	if sendErr != nil {
		updateState(false, sent, sendErr.Error(), lastBarcodeError)
		return result, sendErr
	}
	updateState(true, sent, "", lastBarcodeError)
	return result, nil
}

func (s *NotificationService) sendScenario(
	ctx context.Context,
	cfg dto.NotificationConfig,
	shop *model.MarketplaceShop,
	shopName, scenario string,
	cards []model.AftersaleFilterCard,
	notified map[string]string,
	now time.Time,
) (sent, skipped, barcodeWarnings int, lastBarcodeError string, sendErr error) {
	label := scenarioLabel(scenario, cards)
	group := scenarioGroup(scenario, cards)

	if status := shippedRefundScenarioStatus(scenario); status != "" {
		rows, err := s.shopRepo().ListShippedRefundsByStatus(shop.ID, status)
		if err != nil {
			return 0, 0, 0, "", err
		}
		for _, row := range rows {
			if ClassifyShippedRefundStatus(row.Logistics, row.TrackJSON, row.LogisticsStatus) != status {
				continue
			}
			key := notificationKey(shop.ID, "shipped_refund", row.PlatformAftersaleID, scenario, "")
			if notified[key] != "" {
				skipped++
				continue
			}
			card := buildShippedRefundCard(shopName, label, row)
			if err := s.feishu.SendInteractiveCard(ctx, cfg.WebhookURL, cfg.Secret, card); err != nil {
				return sent, skipped, barcodeWarnings, lastBarcodeError, err
			}
			sent++
			s.markNotified(notified, key)
		}
		return sent, skipped, barcodeWarnings, lastBarcodeError, nil
	}

	if scenario == ScenarioUrgent {
		tickets, err := s.shopRepo().ListTicketsWithDeadline(shop.ID)
		if err != nil {
			return 0, 0, 0, "", err
		}
		for _, t := range tickets {
			urgency := urgencyOf(t.DeadlineAt, now)
			if urgency == "" {
				continue
			}
			key := notificationKey(shop.ID, "ticket", t.PlatformAftersaleID, scenario, urgency)
			if notified[key] != "" {
				skipped++
				continue
			}
			card := buildTicketCard(shopName, "时效紧迫", group, t, now)
			if err := s.attachLogisticsBarcode(ctx, cfg, &card, t.ReturnLogisticsNo); err != nil {
				barcodeWarnings++
				lastBarcodeError = err.Error()
			}
			if err := s.feishu.SendInteractiveCard(ctx, cfg.WebhookURL, cfg.Secret, card); err != nil {
				return sent, skipped, barcodeWarnings, lastBarcodeError, err
			}
			sent++
			s.markNotified(notified, key)
		}
		return sent, skipped, barcodeWarnings, lastBarcodeError, nil
	}

	tickets, err := s.shopRepo().ListTicketsByCard(shop.ID, scenario)
	if err != nil {
		return 0, 0, 0, "", err
	}
	for _, t := range tickets {
		key := notificationKey(shop.ID, "ticket", t.PlatformAftersaleID, scenario, "")
		if notified[key] != "" {
			skipped++
			continue
		}
		card := buildTicketCard(shopName, label, group, t, now)
		if err := s.attachLogisticsBarcode(ctx, cfg, &card, t.ReturnLogisticsNo); err != nil {
			barcodeWarnings++
			lastBarcodeError = err.Error()
		}
		if err := s.feishu.SendInteractiveCard(ctx, cfg.WebhookURL, cfg.Secret, card); err != nil {
			return sent, skipped, barcodeWarnings, lastBarcodeError, err
		}
		sent++
		s.markNotified(notified, key)
	}
	return sent, skipped, barcodeWarnings, lastBarcodeError, nil
}

func (s *NotificationService) markNotified(notified map[string]string, key string) {
	at := time.Now().Format(time.RFC3339)
	notified[key] = at
	_ = s.repos.Notification.UpdateState(s.tenantID, func(st *dto.NotificationState) error {
		if st.Notified == nil {
			st.Notified = map[string]string{}
		}
		st.Notified[key] = at
		return nil
	})
}

func (s *NotificationService) attachLogisticsBarcode(ctx context.Context, cfg dto.NotificationConfig, card *feishu.InteractiveCard, sid string) error {
	sid = strings.TrimSpace(sid)
	if sid == "" {
		return nil
	}
	if cfg.AppID == "" || cfg.AppSecret == "" {
		return fmt.Errorf("未配置飞书应用凭证，无法生成条形码")
	}
	key, err := s.feishu.UploadBarcodeImage(ctx, cfg.AppID, cfg.AppSecret, sid)
	if err != nil {
		return err
	}
	card.FooterImgKey = key
	card.FooterImgAlt = sid
	return nil
}

func (s *NotificationService) ListTenantIDs() []uint64 {
	seen := map[uint64]struct{}{}
	add := func(ids []uint64, err error) {
		if err != nil {
			return
		}
		for _, id := range ids {
			if id == 0 {
				continue
			}
			seen[id] = struct{}{}
		}
	}
	add(s.repos.Notification.ListTenantIDs())
	add(s.repos.Shop.ListTenantIDs())
	out := make([]uint64, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	return out
}

func (s *NotificationService) AnyEnabled() bool {
	for _, tid := range s.ListTenantIDs() {
		if s.ForTenant(tid).Enabled() {
			return true
		}
	}
	return false
}

func (s *NotificationService) MinPollInterval() time.Duration {
	min := 5 * time.Minute
	found := false
	for _, tid := range s.ListTenantIDs() {
		svc := s.ForTenant(tid)
		if !svc.Enabled() {
			continue
		}
		iv := svc.PollInterval()
		if !found || iv < min {
			min = iv
			found = true
		}
	}
	return min
}

func (s *NotificationService) RunAll(ctx context.Context) (sent, skipped int, lastErr error) {
	for _, tid := range s.ListTenantIDs() {
		svc := s.ForTenant(tid)
		if !svc.Enabled() {
			continue
		}
		result, err := svc.RunPoll(ctx, 0)
		if err != nil {
			lastErr = err
			continue
		}
		if result != nil {
			sent += result.Sent
			skipped += result.Skipped
		}
	}
	return sent, skipped, lastErr
}
