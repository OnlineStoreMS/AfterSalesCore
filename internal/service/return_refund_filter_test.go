package service

import "testing"

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
