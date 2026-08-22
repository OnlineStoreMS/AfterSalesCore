package service

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

var timeoutRemainRe = regexp.MustCompile(`(?:(\d+)\s*天)?(?:(\d+)\s*小时)?(?:(\d+)\s*分)?(?:(\d+)\s*秒)?后(.+)`)

// ParseTimeout 从「2天9小时16分后自动同意」解析截止时间和动作，供提醒使用。
func ParseTimeout(text string, from time.Time) (deadline *time.Time, action string, remainSec int) {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\n", ""))
	if text == "" || from.IsZero() {
		return nil, "", 0
	}
	m := timeoutRemainRe.FindStringSubmatch(text)
	if m == nil {
		return nil, "", 0
	}
	days := atoiDefault(m[1])
	hours := atoiDefault(m[2])
	mins := atoiDefault(m[3])
	secs := atoiDefault(m[4])
	action = strings.TrimSpace(m[5])
	action = strings.TrimRight(action, "。.;；")
	dur := time.Duration(days)*24*time.Hour + time.Duration(hours)*time.Hour + time.Duration(mins)*time.Minute + time.Duration(secs)*time.Second
	if dur <= 0 {
		return nil, action, 0
	}
	t := from.Add(dur)
	return &t, action, int(dur.Seconds())
}

func remainSeconds(deadline *time.Time, now time.Time) int {
	if deadline == nil {
		return 0
	}
	sec := int(deadline.Sub(now).Seconds())
	if sec < 0 {
		return 0
	}
	return sec
}

func atoiDefault(s string) int {
	if s == "" {
		return 0
	}
	n, _ := strconv.Atoi(s)
	return n
}
