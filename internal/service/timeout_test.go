package service

import (
	"testing"
	"time"
)

func TestParseTimeout(t *testing.T) {
	from := time.Date(2026, 8, 23, 3, 0, 0, 0, time.Local)
	cases := []struct {
		in     string
		days   int
		hours  int
		mins   int
		secs   int
		action string
	}{
		{"2天9小时16分后自动同意", 2, 9, 16, 0, "自动同意"},
		{"5天17小时52分后售后关闭", 5, 17, 52, 0, "售后关闭"},
		{"6天8小时11分后超时平台将主动发起与用户的换货协商", 6, 8, 11, 0, "超时平台将主动发起与用户的换货协商"},
		{"3小时后自动同意", 0, 3, 0, 0, "自动同意"},
		{"4小时36分54秒后自动同意", 0, 4, 36, 54, "自动同意"},
		{"9小时24分1秒后逾期", 0, 9, 24, 1, "逾期"},
		{"", 0, 0, 0, 0, ""},
		{"同意退款，退款成功", 0, 0, 0, 0, ""},
	}
	for _, tc := range cases {
		deadline, action, remain := ParseTimeout(tc.in, from)
		if action != tc.action {
			t.Fatalf("%q action=%q want %q", tc.in, action, tc.action)
		}
		want := tc.days*86400 + tc.hours*3600 + tc.mins*60 + tc.secs
		if remain != want {
			t.Fatalf("%q remain=%d want %d", tc.in, remain, want)
		}
		if want == 0 {
			if deadline != nil {
				t.Fatalf("%q expected nil deadline", tc.in)
			}
			continue
		}
		if deadline == nil {
			t.Fatalf("%q expected deadline", tc.in)
		}
		if got := int(deadline.Sub(from).Seconds()); got != want {
			t.Fatalf("%q deadline delta=%d want %d", tc.in, got, want)
		}
	}
}

func TestDeadlineFromUnix(t *testing.T) {
	from := time.Date(2026, 8, 23, 3, 0, 0, 0, time.Local)
	deadline, action := deadlineFromUnix(from.Add(9*time.Hour+24*time.Minute).Unix(), "9小时24分后逾期", from)
	if action != "逾期" {
		t.Fatalf("action=%q", action)
	}
	if deadline == nil {
		t.Fatal("expected deadline")
	}
	if got := deadline.Unix(); got != from.Add(9*time.Hour+24*time.Minute).Unix() {
		t.Fatalf("deadline unix=%d", got)
	}
	parsed, action2 := deadlineFromUnix(0, "1小时后逾期", from)
	if action2 != "逾期" || parsed == nil {
		t.Fatalf("fallback parse action=%q deadline=%v", action2, parsed)
	}
}
