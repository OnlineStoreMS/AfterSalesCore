package service

import (
	"testing"
	"time"

	"aftersalescore/internal/model"
	"aftersalescore/internal/repo"
)

func TestAgentCollectDue(t *testing.T) {
	now := time.Date(2026, 9, 7, 8, 0, 0, 0, time.Local)
	past := now.Add(-time.Minute)
	future := now.Add(time.Minute)

	if agentCollectDue(nil, now) {
		t.Fatal("nil shop")
	}
	if agentCollectDue(&model.MarketplaceShop{}, now) {
		t.Fatal("nil next run should not be due")
	}
	if !agentCollectDue(&model.MarketplaceShop{AgentNextRunAt: &past}, now) {
		t.Fatal("past next run should be due")
	}
	if agentCollectDue(&model.MarketplaceShop{AgentNextRunAt: &future}, now) {
		t.Fatal("future next run should not be due")
	}
}

func TestNextSyncHint(t *testing.T) {
	now := time.Date(2026, 9, 7, 8, 0, 0, 0, time.Local)
	interval := 30 * time.Minute
	next := now.Add(interval)
	past := now.Add(-time.Minute)

	cases := []struct {
		name string
		shop *model.MarketplaceShop
		want string
	}{
		{name: "unbound", shop: &model.MarketplaceShop{}, want: ""},
		{
			name: "scheduled",
			shop: &model.MarketplaceShop{PluginKey: "k", AgentNextRunAt: &next},
			want: formatTime(next),
		},
		{
			name: "due requested",
			shop: &model.MarketplaceShop{PluginKey: "k", AgentNextRunAt: &past, SyncRequestedAt: &past},
			want: "已请求，等待 Agent 执行",
		},
		{
			name: "due idle",
			shop: &model.MarketplaceShop{PluginKey: "k", AgentNextRunAt: &past},
			want: "待自动执行",
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

func TestNormalizeRefundApplyRange(t *testing.T) {
	cases := map[string]string{
		"":                          "30",
		"30":                        "30",
		"近30天":                      "30",
		"最近30天":                     "30",
		"全部":                        "all",
		"all":                       "all",
		"7":                         "7",
		"7d":                        "7",
		"近7天":                       "7",
		"最近7天":                      "7",
		"90d":                       "90",
		"近90日":                      "90",
		"最近90天":                     "90",
		"2026-08-01,2026-09-21":     "2026-08-01,2026-09-21",
		"custom:2026/8/1,2026/9/21": "2026-08-01,2026-09-21",
		"2026-09-21,2026-08-01":     "2026-08-01,2026-09-21",
	}
	for in, want := range cases {
		if got := repo.NormalizeRefundApplyRange(in); got != want {
			t.Fatalf("%q => %q, want %q", in, got, want)
		}
	}
}
