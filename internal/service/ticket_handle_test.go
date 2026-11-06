package service

import (
	"testing"

	"aftersalescore/internal/model"
)

func TestNormalizeTicketHandleStatus(t *testing.T) {
	if got := NormalizeTicketHandleStatus(""); got != model.TicketHandlePending {
		t.Fatalf("empty=%q", got)
	}
	if got := NormalizeTicketHandleStatus("已处理"); got != model.TicketHandleHandled {
		t.Fatalf("handled=%q", got)
	}
	if got := NormalizeTicketHandleStatus("已转问题记录"); got != model.TicketHandleIssue {
		t.Fatalf("issue=%q", got)
	}
	if got := NormalizeTicketHandleStatus("unknown"); got != "" {
		t.Fatalf("unknown=%q", got)
	}
}

func TestMatchTicketHandleStatus(t *testing.T) {
	pending := &model.AftersaleTicket{}
	handled := &model.AftersaleTicket{HandleStatus: model.TicketHandleHandled}
	if !matchTicketHandleStatus(pending, "pending") {
		t.Fatal("empty should match 未处理")
	}
	if matchTicketHandleStatus(pending, "handled") {
		t.Fatal("empty should not match 已处理")
	}
	if !matchTicketHandleStatus(handled, "handled") {
		t.Fatal("handled should match")
	}
	if !matchTicketHandleStatus(handled, "") {
		t.Fatal("empty filter should match all")
	}
}
