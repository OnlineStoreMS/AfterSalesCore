package service

import (
	"strings"

	"aftersalescore/internal/dto"
	"aftersalescore/internal/model"
)

const (
	ScenarioUrgent      = "urgent"
	ScenarioAwaitPickup = "shipped_refund_await_pickup"
	ScenarioSigned      = "shipped_refund_signed"
	ScenarioInTransit   = "shipped_refund_in_transit"
)

var fixedScenarioOptions = []dto.ScenarioOption{
	{Key: ScenarioUrgent, Label: "时效紧迫", Group: "时效"},
	{Key: ScenarioAwaitPickup, Label: "待取件", Group: "已发货退款成功"},
	{Key: ScenarioSigned, Label: "已签收", Group: "已发货退款成功"},
	{Key: ScenarioInTransit, Label: "运输中", Group: "已发货退款成功"},
}

func IsAggregateCard(label string) bool {
	s := strings.TrimSpace(label)
	return strings.Contains(s, "全部待收货")
}

func cardScenarioLabel(card model.AftersaleFilterCard) string {
	if card.GroupName != "" && card.CardLabel != "" && !strings.HasPrefix(card.CardLabel, card.GroupName) {
		return card.GroupName + " · " + card.CardLabel
	}
	if card.CardLabel != "" {
		return card.CardLabel
	}
	return card.CardKey
}

func uniqueCardScenarios(cards []model.AftersaleFilterCard) []dto.ScenarioOption {
	seen := map[string]struct{}{}
	out := make([]dto.ScenarioOption, 0)
	for _, c := range cards {
		if c.CardKey == "" || IsAggregateCard(c.CardLabel) || IsAggregateCard(c.CardKey) {
			continue
		}
		if _, ok := seen[c.CardKey]; ok {
			continue
		}
		seen[c.CardKey] = struct{}{}
		out = append(out, dto.ScenarioOption{
			Key:   c.CardKey,
			Label: cardScenarioLabel(c),
			Group: c.GroupName,
		})
	}
	return out
}

func scenarioOptions(cards []model.AftersaleFilterCard) []dto.ScenarioOption {
	out := uniqueCardScenarios(cards)
	out = append(out, fixedScenarioOptions...)
	return out
}

func sanitizeScenarios(keys []string, cards []model.AftersaleFilterCard) []string {
	out := make([]string, 0, len(keys))
	seen := map[string]struct{}{}
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" || !scenarioAllowed(key, cards) {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

func isLegacyServiceScenario(key string) bool {
	return strings.HasPrefix(strings.TrimSpace(key), "service:")
}

func scenarioAllowed(key string, cards []model.AftersaleFilterCard) bool {
	key = strings.TrimSpace(key)
	if key == "" || isLegacyServiceScenario(key) {
		return false
	}
	if IsAggregateCard(key) {
		return false
	}
	for _, opt := range fixedScenarioOptions {
		if opt.Key == key {
			return true
		}
	}
	for _, c := range cards {
		if c.CardKey == key && !IsAggregateCard(c.CardLabel) {
			return true
		}
	}
	return false
}

func scenarioLabel(key string, cards []model.AftersaleFilterCard) string {
	for _, opt := range fixedScenarioOptions {
		if opt.Key == key {
			if opt.Group != "" && opt.Group != "时效" {
				return opt.Group + " · " + opt.Label
			}
			return opt.Label
		}
	}
	for _, c := range cards {
		if c.CardKey == key {
			return cardScenarioLabel(c)
		}
	}
	return key
}

func shippedRefundScenarioStatus(scenario string) string {
	switch scenario {
	case ScenarioAwaitPickup:
		return LogisticsAwaitPickup
	case ScenarioSigned:
		return LogisticsSigned
	case ScenarioInTransit:
		return LogisticsInTransit
	default:
		return ""
	}
}

func scenarioGroup(key string, cards []model.AftersaleFilterCard) string {
	for _, opt := range fixedScenarioOptions {
		if opt.Key == key {
			return opt.Group
		}
	}
	for _, c := range cards {
		if c.CardKey == key {
			return c.GroupName
		}
	}
	return ""
}
