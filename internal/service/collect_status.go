package service

import (
	"strings"
	"time"

	"aftersalescore/internal/model"
)

const (
	CollectStatusDisabled = "disabled"
	CollectStatusOK       = "ok"
	CollectStatusWaiting  = "waiting"
	CollectStatusStale    = "stale"
	CollectStatusError    = "error"
)

func collectStaleAfter(interval time.Duration) time.Duration {
	if interval <= 0 {
		interval = 30 * time.Minute
	}
	wait := interval * 2
	if wait < 2*time.Hour {
		return 2 * time.Hour
	}
	return wait
}

func hasFreshCollectError(shop *model.MarketplaceShop) bool {
	if shop == nil || strings.TrimSpace(shop.CollectError) == "" {
		return false
	}
	if shop.LastSyncAt == nil {
		return true
	}
	if shop.CollectErrorAt == nil {
		return true
	}
	return !shop.CollectErrorAt.Before(*shop.LastSyncAt)
}

func DeriveCollectStatus(shop *model.MarketplaceShop, interval time.Duration, now time.Time) (status, label string) {
	if shop == nil || strings.TrimSpace(shop.PluginKey) == "" {
		return CollectStatusDisabled, "未启用"
	}
	staleAfter := collectStaleAfter(interval)
	if hasFreshCollectError(shop) {
		return CollectStatusError, "采集异常"
	}
	requested := shop.SyncRequestedAt
	waiting := requested != nil && now.Sub(*requested) <= staleAfter &&
		(shop.LastSyncAt == nil || requested.After(*shop.LastSyncAt))
	if waiting {
		return CollectStatusWaiting, "执行中"
	}
	if shop.LastSyncAt == nil {
		if requested != nil {
			return CollectStatusStale, "执行超时"
		}
		return CollectStatusStale, "从未同步"
	}
	if now.Sub(*shop.LastSyncAt) > staleAfter {
		return CollectStatusStale, "同步停滞"
	}
	return CollectStatusOK, "正常"
}

func (s *ShopService) RecordCollectError(shop *model.MarketplaceShop, errMsg string) error {
	if shop == nil || shop.ID == 0 {
		return nil
	}
	msg := strings.TrimSpace(errMsg)
	if msg == "" {
		msg = "采集失败"
	}
	runes := []rune(msg)
	if len(runes) > 1000 {
		msg = string(runes[:1000]) + "…"
	}
	now := time.Now()
	shop.CollectError = msg
	shop.CollectErrorAt = &now
	return s.repos.Shop.ForTenant(shop.TenantID).SaveCollectError(shop.ID, msg, &now)
}
