package service

import (
	"testing"
	"time"

	"aftersalescore/internal/model"
)

func TestDeriveCollectStatus(t *testing.T) {
	now := time.Date(2026, 11, 10, 1, 0, 0, 0, time.Local)
	interval := 30 * time.Minute
	recent := now.Add(-20 * time.Minute)
	old := now.Add(-3 * 24 * time.Hour)
	errAt := now.Add(-10 * time.Minute)

	if st, _ := DeriveCollectStatus(&model.MarketplaceShop{}, interval, now); st != CollectStatusDisabled {
		t.Fatalf("unbound=%s", st)
	}

	okShop := &model.MarketplaceShop{PluginKey: "k", LastSyncAt: &recent}
	if st, label := DeriveCollectStatus(okShop, interval, now); st != CollectStatusOK || label != "正常" {
		t.Fatalf("ok=%s %s", st, label)
	}

	staleShop := &model.MarketplaceShop{PluginKey: "k", LastSyncAt: &old}
	if st, _ := DeriveCollectStatus(staleShop, interval, now); st != CollectStatusStale {
		t.Fatalf("stale=%s", st)
	}

	errShop := &model.MarketplaceShop{
		PluginKey: "k", LastSyncAt: &old, CollectError: "串店登录", CollectErrorAt: &errAt,
	}
	if st, _ := DeriveCollectStatus(errShop, interval, now); st != CollectStatusError {
		t.Fatalf("error=%s", st)
	}

	waitingShop := &model.MarketplaceShop{
		PluginKey: "k", LastSyncAt: &old, SyncRequestedAt: &recent,
	}
	if st, _ := DeriveCollectStatus(waitingShop, interval, now); st != CollectStatusWaiting {
		t.Fatalf("waiting=%s", st)
	}

	cleared := &model.MarketplaceShop{
		PluginKey: "k", LastSyncAt: &recent, CollectError: "旧错误", CollectErrorAt: &old,
	}
	if st, _ := DeriveCollectStatus(cleared, interval, now); st != CollectStatusOK {
		t.Fatalf("old error after later sync should be ok, got %s", st)
	}
}
