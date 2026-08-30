package service

import (
	"testing"
	"time"

	"aftersalescore/internal/model"
	"aftersalescore/internal/repo"
)

func TestPluginShouldSync(t *testing.T) {
	now := time.Date(2026, 8, 23, 8, 0, 0, 0, time.Local)
	interval := 5 * time.Minute
	req := now.Add(-time.Minute)
	recent := now.Add(-2 * time.Minute)
	stale := now.Add(-6 * time.Minute)

	cases := []struct {
		name string
		shop *model.MarketplaceShop
		want bool
	}{
		{name: "nil shop", shop: nil, want: false},
		{name: "never synced", shop: &model.MarketplaceShop{}, want: true},
		{name: "requested", shop: &model.MarketplaceShop{LastSyncAt: &recent, SyncRequestedAt: &req}, want: true},
		{name: "interval not reached", shop: &model.MarketplaceShop{LastSyncAt: &recent}, want: false},
		{name: "interval elapsed", shop: &model.MarketplaceShop{LastSyncAt: &stale}, want: true},
	}
	for _, tc := range cases {
		if got := pluginShouldSync(tc.shop, now, interval); got != tc.want {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}

func TestNextSyncHint(t *testing.T) {
	now := time.Date(2026, 8, 31, 8, 0, 0, 0, time.Local)
	interval := 30 * time.Minute
	online := now.Add(-10 * time.Second)
	offline := now.Add(-10 * time.Minute)
	recentSync := now.Add(-10 * time.Minute)
	staleSync := now.Add(-40 * time.Minute)

	cases := []struct {
		name string
		shop *model.MarketplaceShop
		want string
	}{
		{name: "unbound", shop: &model.MarketplaceShop{}, want: ""},
		{
			name: "online due",
			shop: &model.MarketplaceShop{PluginKey: "k", LastSeenAt: &online, LastSyncAt: &staleSync},
			want: "下一次心跳",
		},
		{
			name: "online waiting",
			shop: &model.MarketplaceShop{PluginKey: "k", LastSeenAt: &online, LastSyncAt: &recentSync},
			want: formatTime(recentSync.Add(interval)),
		},
		{
			name: "offline due",
			shop: &model.MarketplaceShop{PluginKey: "k", LastSeenAt: &offline, LastSyncAt: &staleSync},
			want: "待插件上线后同步",
		},
	}
	for _, tc := range cases {
		if got := nextSyncHint(tc.shop, interval, now); got != tc.want {
			t.Fatalf("%s: got %q want %q", tc.name, got, tc.want)
		}
	}
}

func TestClampPluginSyncMinutes(t *testing.T) {
	if repo.ClampPluginSyncMinutes(0) != 30 {
		t.Fatal("unset should default to 30")
	}
	if repo.ClampPluginSyncMinutes(2) != 5 {
		t.Fatal("below min should be 5")
	}
	if repo.ClampPluginSyncMinutes(2000) != 1440 {
		t.Fatal("above max should be 1440")
	}
	if repo.ClampPluginSyncMinutes(60) != 60 {
		t.Fatal("60 should stay 60")
	}
}
