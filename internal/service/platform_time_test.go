package service

import (
	"testing"
	"time"
)

func TestParsePlatformDateTime(t *testing.T) {
	got := ParsePlatformDateTime("2026/08/27 16:55:30", 0)
	if got == nil {
		t.Fatal("expected apply time")
	}
	if got.Year() != 2026 || got.Month() != time.August || got.Day() != 27 || got.Hour() != 16 {
		t.Fatalf("got %v", got)
	}

	track := ParsePlatformDateTime("08/13 09:56:03", 2026)
	if track == nil {
		t.Fatal("expected track time")
	}
	if track.Year() != 2026 || track.Month() != time.August || track.Day() != 13 || track.Hour() != 9 {
		t.Fatalf("got %v", track)
	}
}

func TestResolveReturnTimesFromTrack(t *testing.T) {
	display, returnedAt, appliedAt := ResolveReturnTimes(
		"",
		"2026/08/16 12:34:27",
		"2026/07/30 12:50:37",
		`[{"date":"08/13 09:56:03","title":"已退回","text":"已退回 宿迁宿城"}]`,
	)
	if appliedAt == nil || appliedAt.Day() != 16 {
		t.Fatalf("appliedAt %v", appliedAt)
	}
	if returnedAt == nil || returnedAt.Day() != 13 || returnedAt.Year() != 2026 {
		t.Fatalf("returnedAt %v", returnedAt)
	}
	if display == "" {
		t.Fatal("expected display return time")
	}
}

func TestParseQueryDateTimeEndOfDay(t *testing.T) {
	got := ParseQueryDateTime("2026-08-30", true)
	if got == nil || got.Hour() != 23 || got.Minute() != 59 {
		t.Fatalf("got %v", got)
	}
}
