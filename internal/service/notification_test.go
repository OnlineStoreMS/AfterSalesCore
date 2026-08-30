package service

import (
	"testing"
	"time"

	"aftersalescore/internal/model"
)

func TestIsAggregateCard(t *testing.T) {
	if !IsAggregateCard("全部待收货/发货") {
		t.Fatal("expected aggregate")
	}
	if !IsAggregateCard("全部待收货") {
		t.Fatal("expected aggregate")
	}
	if IsAggregateCard("退货待收货") {
		t.Fatal("subset card should be kept")
	}
	if IsAggregateCard("换货待收货/发货") {
		t.Fatal("subset card should be kept")
	}
}

func TestUniqueCardScenariosSkipsAggregate(t *testing.T) {
	cards := []model.AftersaleFilterCard{
		{GroupName: "待收货/发货", CardKey: "待收货/发货:全部待收货/发货", CardLabel: "全部待收货/发货"},
		{GroupName: "待收货/发货", CardKey: "待收货/发货:退货待收货", CardLabel: "退货待收货"},
		{GroupName: "待收货/发货", CardKey: "待收货/发货:换货待收货/发货", CardLabel: "换货待收货/发货"},
		{GroupName: "紧急", CardKey: "紧急:临期待处理", CardLabel: "临期待处理"},
	}
	opts := uniqueCardScenarios(cards)
	keys := map[string]bool{}
	for _, o := range opts {
		keys[o.Key] = true
	}
	if keys["待收货/发货:全部待收货/发货"] {
		t.Fatal("aggregate card should be skipped")
	}
	if !keys["待收货/发货:退货待收货"] || !keys["待收货/发货:换货待收货/发货"] || !keys["紧急:临期待处理"] {
		t.Fatalf("subset cards missing: %+v", opts)
	}
}

func TestSanitizeScenariosDropsServiceAndAggregate(t *testing.T) {
	cards := []model.AftersaleFilterCard{
		{GroupName: "待商家收/发货", CardKey: "待商家收/发货:退货待收货", CardLabel: "退货待收货"},
		{GroupName: "待商家收/发货", CardKey: "待商家收/发货:全部待收货/发货", CardLabel: "全部待收货/发货"},
	}
	got := sanitizeScenarios([]string{
		"service:待处理",
		"待商家收/发货:退货待收货",
		"待商家收/发货:全部待收货/发货",
		"urgent",
		"urgent",
	}, cards)
	if len(got) != 2 || got[0] != "待商家收/发货:退货待收货" || got[1] != "urgent" {
		t.Fatalf("got %+v", got)
	}
}

func TestPruneServiceNotified(t *testing.T) {
	m := map[string]string{
		"2:service:SO1:service:待处理": "2026-08-01T00:00:00Z",
		"2:ticket:147:service:待处理":  "2026-08-01T00:00:00Z",
		"2:ticket:147:urgent:warning": "2026-08-01T00:00:00Z",
	}
	if !pruneServiceNotified(m) {
		t.Fatal("expected prune")
	}
	if _, ok := m["2:ticket:147:urgent:warning"]; !ok {
		t.Fatal("ticket urgent key should stay")
	}
	if len(m) != 1 {
		t.Fatalf("got %+v", m)
	}
}

func TestNotificationKeyUrgentEscalation(t *testing.T) {
	k1 := notificationKey(2, "ticket", "147", ScenarioUrgent, "warning")
	k2 := notificationKey(2, "ticket", "147", ScenarioUrgent, "critical")
	k3 := notificationKey(2, "ticket", "147", ScenarioUrgent, "imminent")
	k4 := notificationKey(2, "ticket", "147", ScenarioUrgent, "expired")
	keys := []string{k1, k2, k3, k4}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] == keys[j] {
				t.Fatalf("urgency keys must differ: %q vs %q", keys[i], keys[j])
			}
		}
	}
	other := notificationKey(2, "ticket", "147", "紧急:临期待处理", "")
	if other == k4 {
		t.Fatal("card scenario key should differ from urgent")
	}
}

func TestUrgencyOf(t *testing.T) {
	now := time.Date(2026, 8, 23, 5, 0, 0, 0, time.Local)
	expired := now.Add(-time.Minute)
	if urgencyOf(&expired, now) != "expired" {
		t.Fatal("expired")
	}
	soon := now.Add(10 * time.Minute)
	if urgencyOf(&soon, now) != "imminent" {
		t.Fatal("imminent")
	}
	crit := now.Add(2 * time.Hour)
	if urgencyOf(&crit, now) != "critical" {
		t.Fatal("critical")
	}
	warn := now.Add(8 * time.Hour)
	if urgencyOf(&warn, now) != "warning" {
		t.Fatal("warning")
	}
	later := now.Add(2 * 24 * time.Hour)
	if urgencyOf(&later, now) != "" {
		t.Fatal("not urgent")
	}
}

func TestMdLineSkipsEmpty(t *testing.T) {
	if mdLine("买家", "", "") != "" {
		t.Fatal("empty value should skip")
	}
	got := mdLine("退货物流", "YT123", "")
	if got != "**退货物流：** YT123" {
		t.Fatalf("got %q", got)
	}
}
