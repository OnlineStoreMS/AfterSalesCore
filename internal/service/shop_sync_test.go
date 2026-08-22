package service

import (
	"testing"
	"time"

	"aftersalescore/internal/model"
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
