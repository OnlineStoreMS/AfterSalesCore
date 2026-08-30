package service

import (
	"encoding/json"
	"strings"
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
	if len(tracks) > 5 {
		tracks = tracks[:5]
	}
	return tracks
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
	if strings.Contains(s, LogisticsShipped) {
		return LogisticsShipped
	}
	return ""
}

func ClassifyLogisticsStatus(text string) string {
	return matchLogisticsKeyword(text)
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
