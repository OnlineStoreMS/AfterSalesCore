package service

import (
	"testing"

	"aftersalescore/internal/model"
)

func TestIsShippedRefundSuccessRecord(t *testing.T) {
	cases := []struct {
		typ string
		ok  bool
	}{
		{"已发货退款", true},
		{"仅退款已发货退款", true},
		{"未发货退款", false},
		{"已发货退款/未发货退款", false},
		{"退货退款", false},
		{"退货退款成功", false},
		{"仅退款", false},
		{"", false},
	}
	for _, c := range cases {
		got := IsShippedRefundSuccessRecord(c.typ)
		if got != c.ok {
			t.Fatalf("type=%q got %v want %v", c.typ, got, c.ok)
		}
	}
}

func TestIsReturnRefundSuccessRecord(t *testing.T) {
	cases := []struct {
		typ, status string
		ok          bool
	}{
		{"退货退款", "同意退款，退款成功", true},
		{"退货退款", "退款成功", true},
		{"", "同意退款，退款成功", true},
		{"未发货退款", "同意退款，退款成功", false},
		{"已发货退款", "同意退款，退款成功", false},
		{"退货退款", "待商家收货", false},
		{"已发货退款", "拒绝售后申请", false},
		{"换货", "退款成功", false},
		{"退货退款", "", false},
	}
	for _, c := range cases {
		got := IsReturnRefundSuccessRecord(c.typ, c.status)
		if got != c.ok {
			t.Fatalf("type=%q status=%q got %v want %v", c.typ, c.status, got, c.ok)
		}
	}
}

func TestShippedRefundFromReturnKeepsReturned(t *testing.T) {
	got, ok := shippedRefundFromReturn(&model.ReturnPackage{
		PlatformAftersaleID: "A1",
		AftersaleType:       "已发货退款",
		Status:              "已退回",
		Logistics:           "已退回",
		LogisticsNo:         "SF1",
	})
	if !ok || got.LogisticsStatus != LogisticsReturned || got.LogisticsNo != "SF1" {
		t.Fatalf("returned shipped refund = %+v ok=%v", got, ok)
	}
	if _, ok := shippedRefundFromReturn(&model.ReturnPackage{
		PlatformAftersaleID: "A2",
		AftersaleType:       "退货退款",
		Status:              "已退回",
	}); ok {
		t.Fatal("退货退款 should stay out of 已发货退款")
	}
}
