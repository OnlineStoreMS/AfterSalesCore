package service

import (
	"encoding/json"
	"strings"
	"time"
)

func ParsePlatformDateTime(raw string, yearHint int) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	raw = strings.ReplaceAll(raw, "T", " ")
	layouts := []string{
		"2006/01/02 15:04:05",
		"2006-01-02 15:04:05",
		"2006/01/02 15:04",
		"2006-01-02 15:04",
		"2006/01/02",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			return &t
		}
	}
	if yearHint <= 0 {
		yearHint = time.Now().Year()
	}
	for _, layout := range []string{"01/02 15:04:05", "01/02 15:04", "1/2 15:04:05"} {
		if t, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			out := time.Date(yearHint, t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.Local)
			return &out
		}
	}
	return nil
}

func yearHintFrom(applyTime, shipTime string) int {
	if t := ParsePlatformDateTime(applyTime, 0); t != nil {
		return t.Year()
	}
	if t := ParsePlatformDateTime(shipTime, 0); t != nil {
		return t.Year()
	}
	return time.Now().Year()
}

type trackLine struct {
	Date  string `json:"date"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

func returnTimeFromTracks(trackJSON string) string {
	raw := strings.TrimSpace(trackJSON)
	if raw == "" {
		return ""
	}
	var tracks []trackLine
	if err := json.Unmarshal([]byte(raw), &tracks); err != nil || len(tracks) == 0 {
		return ""
	}
	for _, t := range tracks {
		if strings.Contains(t.Title, "已退回") || strings.Contains(t.Text, "已退回") {
			if strings.TrimSpace(t.Date) != "" {
				return t.Date
			}
		}
	}
	return strings.TrimSpace(tracks[0].Date)
}

func ResolveReturnTimes(returnTime, applyTime, shipTime, trackJSON string) (displayReturn string, returnedAt, appliedAt *time.Time) {
	appliedAt = ParsePlatformDateTime(applyTime, 0)
	hint := yearHintFrom(applyTime, shipTime)
	displayReturn = strings.TrimSpace(returnTime)
	if displayReturn != "" {
		returnedAt = ParsePlatformDateTime(displayReturn, hint)
	}
	if returnedAt == nil {
		if date := returnTimeFromTracks(trackJSON); date != "" {
			returnedAt = ParsePlatformDateTime(date, hint)
			if displayReturn == "" && returnedAt != nil {
				displayReturn = returnedAt.Format("2006/01/02 15:04:05")
			} else if displayReturn == "" {
				displayReturn = date
			}
		}
	}
	if displayReturn == "" && returnedAt != nil {
		displayReturn = returnedAt.Format("2006/01/02 15:04:05")
	}
	return displayReturn, returnedAt, appliedAt
}

func ParseQueryDateTime(raw string, endOfDay bool) *time.Time {
	t := ParsePlatformDateTime(raw, 0)
	if t == nil {
		return nil
	}
	if len(strings.TrimSpace(raw)) <= 10 && endOfDay {
		eod := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, t.Location())
		return &eod
	}
	return t
}
