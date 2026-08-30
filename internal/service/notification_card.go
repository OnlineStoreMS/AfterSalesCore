package service

import (
	"fmt"
	"strings"
	"time"

	"aftersalescore/internal/feishu"
	"aftersalescore/internal/model"
)

func buildTicketCard(shopName, scenarioLabel, group string, t model.AftersaleTicket, now time.Time) feishu.InteractiveCard {
	urgency := urgencyOf(t.DeadlineAt, now)
	var lines []string
	if line := mdLine("店铺", shopName, "blue"); line != "" {
		lines = append(lines, line)
	}
	if line := mdLine("订单号", t.OrderNo, ""); line != "" {
		lines = append(lines, line)
	}
	if line := mdLine("售后单", t.PlatformAftersaleID, ""); line != "" {
		lines = append(lines, line)
	}
	if line := mdLine("类型", t.AftersaleType, "purple"); line != "" {
		lines = append(lines, line)
	}
	if line := mdLine("状态", t.Status, ""); line != "" {
		lines = append(lines, line)
	}
	if line := mdLine("退货物流", t.ReturnLogisticsNo, ""); line != "" {
		lines = append(lines, line)
	}
	if line := mdLine("商品", t.ProductTitle, ""); line != "" {
		lines = append(lines, line)
	}
	if line := mdLine("规格", t.SKU, ""); line != "" {
		lines = append(lines, line)
	}
	if remain := formatRemainText(t.DeadlineAt, t.TimeoutText, t.TimeoutAction, now); remain != "" {
		if line := mdLine("时效", remain, urgencyColor(urgency)); line != "" {
			lines = append(lines, line)
		}
	}
	if line := mdLine("说明", firstNonEmpty(t.TimeoutText, t.Reason), "grey"); line != "" {
		lines = append(lines, line)
	}
	if line := mdLine("物流", truncateText(t.Logistics, 120), "grey"); line != "" {
		lines = append(lines, line)
	}
	return feishu.InteractiveCard{
		Title:    "售后通知 · " + scenarioLabel,
		Template: scenarioCardTemplate(scenarioLabel, group, urgency),
		Markdown: strings.Join(lines, "\n"),
	}
}

func buildShippedRefundCard(shopName, scenarioLabel string, row model.ShippedRefundSuccess) feishu.InteractiveCard {
	var lines []string
	if line := mdLine("店铺", shopName, "blue"); line != "" {
		lines = append(lines, line)
	}
	if line := mdLine("物流状态", firstNonEmpty(row.LogisticsStatus, ClassifyLogisticsStatus(row.Logistics)), "red"); line != "" {
		lines = append(lines, line)
	}
	if line := mdLine("订单号", row.OrderNo, ""); line != "" {
		lines = append(lines, line)
	}
	if line := mdLine("售后单", row.PlatformAftersaleID, ""); line != "" {
		lines = append(lines, line)
	}
	if line := mdLine("类型", firstNonEmpty(row.AftersaleType, "已发货退款"), "purple"); line != "" {
		lines = append(lines, line)
	}
	if line := mdLine("状态", row.Status, ""); line != "" {
		lines = append(lines, line)
	}
	if line := mdLine("商品", row.ProductTitle, ""); line != "" {
		lines = append(lines, line)
	}
	if line := mdLine("规格", row.SKU, ""); line != "" {
		lines = append(lines, line)
	}
	if line := mdLine("订单信息", truncateText(row.OrderInfo, 220), ""); line != "" {
		lines = append(lines, line)
	}
	if line := mdLine("售后信息", truncateText(row.AftersaleInfo, 220), ""); line != "" {
		lines = append(lines, line)
	}
	if line := mdLine("物流单号", firstNonEmpty(row.LogisticsNo, ""), ""); line != "" {
		lines = append(lines, line)
	}
	if line := mdLine("物流", truncateText(row.Logistics, 160), "red"); line != "" {
		lines = append(lines, line)
	}
	for i, tr := range ParseLogisticsTracks(row.TrackJSON) {
		label := "轨迹"
		if i == 0 {
			label = "最近轨迹"
		}
		if line := mdLine(label, truncateText(firstNonEmpty(tr.Text, tr.Title), 160), ""); line != "" {
			lines = append(lines, line)
		}
	}
	return feishu.InteractiveCard{
		Title:    "售后通知 · " + scenarioLabel,
		Template: "red",
		Markdown: strings.Join(lines, "\n"),
	}
}

func urgencyOf(deadline *time.Time, now time.Time) string {
	if deadline == nil {
		return ""
	}
	sec := int(deadline.Sub(now).Seconds())
	if sec <= 0 {
		return "expired"
	}
	if sec <= 30*60 {
		return "imminent"
	}
	if sec <= 4*3600 {
		return "critical"
	}
	if sec <= 12*3600 {
		return "warning"
	}
	return ""
}

func urgencyColor(urgency string) string {
	switch urgency {
	case "expired", "imminent", "critical":
		return "red"
	case "warning":
		return "orange"
	default:
		return "green"
	}
}

func scenarioCardTemplate(label, group, urgency string) string {
	switch urgency {
	case "expired", "imminent", "critical":
		return "red"
	case "warning":
		return "orange"
	}
	if strings.Contains(group, "紧急") || strings.Contains(label, "临期") || strings.Contains(label, "已逾期") {
		return "red"
	}
	if strings.Contains(label, "换货") {
		return "purple"
	}
	if strings.Contains(label, "退货待收货") {
		return "green"
	}
	if strings.Contains(label, "待处理") || strings.Contains(label, "待商家") {
		return "orange"
	}
	return "blue"
}

func formatRemainText(deadline *time.Time, timeoutText, action string, now time.Time) string {
	if deadline != nil {
		sec := int(deadline.Sub(now).Seconds())
		if sec <= 0 {
			if action != "" {
				return "已超时 · " + action
			}
			return "已超时"
		}
		text := "剩余 " + formatRemain(sec)
		if action != "" {
			text += "后" + action
		}
		return text
	}
	return strings.TrimSpace(timeoutText)
}

func formatRemain(sec int) string {
	if sec <= 0 {
		return "已超时"
	}
	d := sec / 86400
	h := (sec % 86400) / 3600
	m := (sec % 3600) / 60
	s := sec % 60
	var parts []string
	if d > 0 {
		parts = append(parts, fmt.Sprintf("%d天", d))
	}
	if h > 0 {
		parts = append(parts, fmt.Sprintf("%d小时", h))
	}
	if d > 0 || h > 0 || m > 0 {
		parts = append(parts, fmt.Sprintf("%d分", m))
	}
	if d == 0 && h < 6 {
		parts = append(parts, fmt.Sprintf("%d秒", s))
	}
	if len(parts) == 0 {
		return "不足1分"
	}
	return strings.Join(parts, "")
}

func mdLine(label, value, color string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = escapeLarkMD(value)
	if color != "" {
		value = fmt.Sprintf("<font color='%s'>%s</font>", color, value)
	}
	return fmt.Sprintf("**%s：** %s", escapeLarkMD(label), value)
}

func escapeLarkMD(s string) string {
	return strings.NewReplacer("<", "&lt;", ">", "&gt;").Replace(s)
}

func truncateText(s string, max int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= max {
		return string(r)
	}
	return string(r[:max]) + "..."
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func notificationKey(shopID uint64, kind, id, scenario, urgency string) string {
	if scenario == ScenarioUrgent && urgency != "" {
		return fmt.Sprintf("%d:%s:%s:urgent:%s", shopID, kind, id, urgency)
	}
	return fmt.Sprintf("%d:%s:%s:%s", shopID, kind, id, scenario)
}

func shopDisplayName(shop *model.MarketplaceShop) string {
	if shop == nil {
		return ""
	}
	if strings.TrimSpace(shop.Name) != "" {
		return shop.Name
	}
	return shop.PlatformShopName
}
