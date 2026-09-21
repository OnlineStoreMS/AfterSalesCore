package service

import (
	"encoding/json"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	LogisticsAwaitPickup = "待取件"
	LogisticsSigned      = "已签收"
	LogisticsInTransit   = "运输中"
	LogisticsReturned    = "已退回"
	LogisticsCancelled   = "已取消"
	LogisticsShipped     = "已发货"
)

type LogisticsTrack struct {
	Date   string `json:"date"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Text   string `json:"text"`
}

func ParseLogisticsTracks(raw string) []LogisticsTrack {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var tracks []LogisticsTrack
	if err := json.Unmarshal([]byte(raw), &tracks); err != nil || len(tracks) == 0 {
		return nil
	}
	return normalizeLogisticsTracks(tracks)
}

var (
	trackTimeRe  = regexp.MustCompile(`(\d{2}/\d{2}\s+\d{2}:\d{2}(?::\d{2})?)`)
	trackTitleRe = regexp.MustCompile(`^(已签收|待取件|运输中|已发货|已揽件|派件中|已退回|已取消)`)
)

func collapseDuplicatedText(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) < 24 {
		return s
	}
	runes := []rune(s)
	almostSame := func(a, b string) bool {
		if a == "" || b == "" {
			return false
		}
		x := strings.TrimRight(a, "。！!．. ")
		y := strings.TrimRight(b, "。！!．. ")
		if x == y && utf8.RuneCountInString(x) >= 12 {
			return true
		}
		longer, shorter := x, y
		if len(y) > len(x) {
			longer, shorter = y, x
		}
		return utf8.RuneCountInString(shorter) >= 12 &&
			strings.HasPrefix(longer, shorter) &&
			len(shorter)*100/len(longer) >= 85
	}
	maxSplit := len(runes) - 12
	if capSplit := len(runes) * 3 / 5; maxSplit > capSplit {
		maxSplit = capSplit
	}
	for lenChars := maxSplit; lenChars >= 12; lenChars-- {
		a := strings.TrimSpace(string(runes[:lenChars]))
		b := strings.TrimSpace(string(runes[lenChars:]))
		if almostSame(a, b) {
			if utf8.RuneCountInString(a) >= utf8.RuneCountInString(b) {
				return a
			}
			return b
		}
	}
	return s
}

func trackDateOf(raw string) string {
	m := trackTimeRe.FindStringSubmatch(raw)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

func trackTitleOf(raw string) string {
	s := strings.Join(strings.Fields(raw), " ")
	if m := trackTitleRe.FindStringSubmatch(s); len(m) > 1 {
		return m[1]
	}
	if utf8.RuneCountInString(s) <= 8 {
		return s
	}
	return ""
}

func stripTrackPrefix(detail, date, title string) string {
	s := strings.Join(strings.Fields(detail), " ")
	for _, part := range []string{date, title} {
		part = strings.Join(strings.Fields(strings.TrimSpace(part)), " ")
		if part != "" && strings.HasPrefix(s, part) {
			s = strings.TrimSpace(s[len(part):])
		}
	}
	return s
}

func normalizeLogisticsTrack(t LogisticsTrack) LogisticsTrack {
	date := trackDateOf(t.Date)
	if date == "" {
		date = trackDateOf(t.Text)
	}
	title := trackTitleOf(t.Title)
	if title == "" {
		title = trackTitleOf(t.Text)
	}
	if title == "" {
		title = trackTitleOf(t.Detail)
	}
	detail := strings.TrimSpace(t.Detail)
	if detail == "" {
		detail = t.Text
	}
	detail = collapseDuplicatedText(stripTrackPrefix(detail, date, title))
	if detail == title || detail == date {
		detail = ""
	}
	text := strings.Join(strings.Fields(strings.Join([]string{date, title, detail}, " ")), " ")
	return LogisticsTrack{Date: date, Title: title, Detail: detail, Text: text}
}

func normalizeLogisticsTracks(tracks []LogisticsTrack) []LogisticsTrack {
	out := make([]LogisticsTrack, 0, len(tracks))
	index := map[string]int{}
	for _, raw := range tracks {
		t := normalizeLogisticsTrack(raw)
		if t.Date == "" && t.Title == "" && t.Detail == "" && t.Text == "" {
			continue
		}
		key := t.Date + "\x00" + t.Title
		if t.Date == "" && t.Title == "" {
			key = t.Text
		}
		if i, ok := index[key]; ok {
			if len(t.Detail) > len(out[i].Detail) {
				out[i].Detail = t.Detail
				out[i].Text = strings.Join(strings.Fields(strings.Join([]string{out[i].Date, out[i].Title, out[i].Detail}, " ")), " ")
			}
			continue
		}
		if len(out) >= 5 {
			continue
		}
		index[key] = len(out)
		out = append(out, t)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func matchLogisticsKeyword(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if strings.Contains(s, LogisticsReturned) {
		return LogisticsReturned
	}
	if strings.Contains(s, LogisticsCancelled) {
		return LogisticsCancelled
	}
	if strings.Contains(s, LogisticsAwaitPickup) {
		return LogisticsAwaitPickup
	}
	if strings.Contains(s, LogisticsSigned) {
		return LogisticsSigned
	}
	if strings.Contains(s, LogisticsInTransit) {
		return LogisticsInTransit
	}
	if strings.Contains(s, "派件中") || strings.Contains(s, "已揽件") {
		return LogisticsInTransit
	}
	if strings.Contains(s, LogisticsShipped) {
		return LogisticsShipped
	}
	return ""
}

func ClassifyLogisticsStatus(text string) string {
	return matchLogisticsKeyword(text)
}

func LatestTrackStatus(trackJSON string) string {
	tracks := ParseLogisticsTracks(trackJSON)
	if len(tracks) == 0 {
		return ""
	}
	return matchLogisticsKeyword(firstNonEmpty(tracks[0].Title, tracks[0].Text))
}

func ClassifyLogisticsWithTracks(logistics, trackJSON string) string {
	if strings.Contains(logistics, LogisticsCancelled) {
		return LogisticsCancelled
	}
	tracks := ParseLogisticsTracks(trackJSON)
	if len(tracks) > 0 {
		latest := firstNonEmpty(tracks[0].Title, tracks[0].Text)
		if s := matchLogisticsKeyword(latest); s != "" && s != LogisticsShipped {
			return s
		}
	}
	var b strings.Builder
	b.WriteString(logistics)
	for _, t := range tracks {
		b.WriteByte('\n')
		b.WriteString(t.Title)
		b.WriteByte(' ')
		b.WriteString(t.Text)
	}
	return matchLogisticsKeyword(b.String())
}

func IsLogisticsAlert(status string) bool {
	return status == LogisticsAwaitPickup || status == LogisticsSigned || status == LogisticsInTransit
}

// ClassifyShippedRefundStatus 已发货退款成功以「订单发货」行为准。
// 买家退货待取件不能盖过订单发货已签收/已发货，否则拦截列表会把退货待取件误收进来。
func ClassifyShippedRefundStatus(logistics, trackJSON, stored string) string {
	if strings.Contains(logistics, LogisticsCancelled) {
		return LogisticsCancelled
	}
	view := ParseTicketLogistics(logistics)
	if view.HasShip && view.ShipStatus != "" {
		return view.ShipStatus
	}
	status := strings.TrimSpace(stored)
	if status == "" || status == LogisticsShipped {
		if better := ClassifyLogisticsWithTracks(logistics, trackJSON); better != "" {
			status = better
		} else if view.HasShip && !view.HasBuyer {
			status = LogisticsShipped
		}
	}
	if isBuyerPickupOnly(view, status) {
		return firstNonEmpty(view.ShipStatus, LogisticsShipped)
	}
	return status
}

func isBuyerPickupOnly(view TicketLogisticsView, status string) bool {
	if status != LogisticsAwaitPickup && view.BuyerStatus != LogisticsAwaitPickup {
		return false
	}
	if view.Intercept || view.ShipStatus == LogisticsAwaitPickup {
		return false
	}
	return view.HasBuyer && view.BuyerStatus == LogisticsAwaitPickup
}

// IsMerchantInterceptPickup 需商家拦截：文案含拦截，或订单发货待取件。
// 仅买家退货待取件不算。
func IsMerchantInterceptPickup(logistics, storedStatus string) bool {
	view := ParseTicketLogistics(logistics)
	if view.Intercept || view.ShipStatus == LogisticsAwaitPickup {
		return true
	}
	if view.HasBuyer {
		return false
	}
	status := ClassifyShippedRefundStatus(logistics, "", storedStatus)
	return status == LogisticsAwaitPickup
}

type TicketLogisticsView struct {
	HasBuyer    bool
	BuyerStatus string
	HasShip     bool
	ShipStatus  string
	Intercept   bool
}

func chunkAfterLabel(raw, label string, stops ...string) string {
	idx := strings.Index(raw, label)
	if idx < 0 {
		return ""
	}
	rest := raw[idx+len(label):]
	cut := len(rest)
	for _, stop := range stops {
		if i := strings.Index(rest, stop); i >= 0 && i < cut {
			cut = i
		}
	}
	return rest[:cut]
}

func ParseTicketLogistics(raw string) TicketLogisticsView {
	text := strings.TrimSpace(raw)
	view := TicketLogisticsView{
		HasBuyer:  strings.Contains(text, "买家退货"),
		HasShip:   strings.Contains(text, "订单发货"),
		Intercept: strings.Contains(text, "订单发货") && strings.Contains(text, "需商家拦截快递"),
	}
	if view.HasBuyer {
		view.BuyerStatus = matchLogisticsKeyword(chunkAfterLabel(text, "买家退货", "订单发货", "需商家拦截快递"))
	}
	if view.HasShip {
		view.ShipStatus = matchLogisticsKeyword(chunkAfterLabel(text, "订单发货", "买家退货", "需商家拦截快递"))
	}
	return view
}

func LimitLogisticsTracksJSON(raw string) string {
	tracks := ParseLogisticsTracks(raw)
	if len(tracks) == 0 {
		return strings.TrimSpace(raw)
	}
	b, err := json.Marshal(tracks)
	if err != nil {
		return strings.TrimSpace(raw)
	}
	return string(b)
}
