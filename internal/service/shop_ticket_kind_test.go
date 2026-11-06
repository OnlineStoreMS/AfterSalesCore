package service

import (
	"testing"
	"time"

	"aftersalescore/internal/dto"
	"aftersalescore/internal/model"
)

func TestMatchShopTicketKind(t *testing.T) {
	pickup := &model.AftersaleTicket{
		Logistics: "买家退货 1/1 待取件\n订单发货 1/1 已发货",
		CardKeys: []model.AftersaleTicketCard{
			{CardKey: "待商家收/发货:待商家收货"},
		},
	}
	if !MatchShopTicketKind(pickup, dto.TicketKindBuyerReturnPickup) {
		t.Fatal("buyer return await-pickup should match")
	}
	if MatchShopTicketKind(pickup, dto.TicketKindReviewShippedRefund) {
		t.Fatal("await-pickup should not match shipped refund review")
	}

	transit := &model.AftersaleTicket{
		Logistics: "买家退货 运输中",
		CardKeys:  []model.AftersaleTicketCard{{CardKey: "待商家收/发货:待商家收货"}},
	}
	if MatchShopTicketKind(transit, dto.TicketKindBuyerReturnPickup) {
		t.Fatal("in-transit buyer return should not match pickup")
	}

	review := &model.AftersaleTicket{
		Logistics: "订单发货 已发货",
		CardKeys:  []model.AftersaleTicketCard{{CardKey: "待商家审核:已发货退款"}},
	}
	if !MatchShopTicketKind(review, dto.TicketKindReviewShippedRefund) {
		t.Fatal("review shipped refund should match")
	}
	if MatchShopTicketKind(review, dto.TicketKindBuyerReturnPickup) {
		t.Fatal("review shipped refund should not match pickup")
	}

	if MatchShopTicketKind(nil, dto.TicketKindBuyerReturnPickup) {
		t.Fatal("nil ticket should not match")
	}

	signed := &model.AftersaleTicket{
		Logistics: "买家退货 1/1 已签收\n订单发货 1/1 已发货",
		CardKeys: []model.AftersaleTicketCard{
			{CardKey: "待商家收/发货:全部待收货/发货"},
		},
	}
	if !MatchShopTicketKind(signed, dto.TicketKindBuyerReturnSigned) {
		t.Fatal("buyer return signed should match")
	}
	if MatchShopTicketKind(pickup, dto.TicketKindBuyerReturnSigned) {
		t.Fatal("await-pickup should not match signed")
	}
	if MatchShopTicketKind(signed, dto.TicketKindBuyerReturnPickup) {
		t.Fatal("signed should not match pickup")
	}

	signedOtherCard := &model.AftersaleTicket{
		Logistics: "买家退货 已签收",
		CardKeys:  []model.AftersaleTicketCard{{CardKey: "待商家收/发货:退货待收货"}},
	}
	if !MatchShopTicketKind(signedOtherCard, dto.TicketKindBuyerReturnSigned) {
		t.Fatal("signed on 退货待收货 should match")
	}

	signedStuckOnReview := &model.AftersaleTicket{
		Logistics: "买家退货 已签收\n订单发货 已签收",
		TrackJSON: `[{"date":"11/05 18:41:18","title":"已签收","detail":"客户成功从丰巢取出快件"}]`,
		CardKeys:  []model.AftersaleTicketCard{{CardKey: "待商家审核:全部待审核"}},
	}
	if !MatchShopTicketKind(signedStuckOnReview, dto.TicketKindBuyerReturnSigned) {
		t.Fatal("signed buyer return stuck on 待商家审核 should still match 退货已签收")
	}

	stalePickup := &model.AftersaleTicket{
		Logistics: "买家退货 待取件\n订单发货 已签收",
		TrackJSON: `[{"title":"已签收","text":"09/09 23:18:34 已签收 快件已领取"},{"title":"待取件","text":"09/08 10:57:13 待取件"}]`,
		CardKeys: []model.AftersaleTicketCard{
			{CardKey: "待商家收/发货:全部待收货/发货"},
			{CardKey: "待商家收/发货:退货待收货"},
		},
	}
	if MatchShopTicketKind(stalePickup, dto.TicketKindBuyerReturnPickup) {
		t.Fatal("stale 待取件 text with latest track 已签收 should leave pickup")
	}
	if !MatchShopTicketKind(stalePickup, dto.TicketKindBuyerReturnSigned) {
		t.Fatal("stale 待取件 text with latest track 已签收 should match signed")
	}

	dispute := &model.AftersaleTicket{
		CardKeys: []model.AftersaleTicketCard{{CardKey: "纠纷:仲裁待举证"}},
	}
	if !MatchShopTicketKind(dispute, dto.TicketKindDispute) {
		t.Fatal("dispute card should match")
	}
	if MatchShopTicketKind(review, dto.TicketKindDispute) {
		t.Fatal("shipped refund should not match dispute")
	}

	returnWaitReceive := &model.AftersaleTicket{
		AftersaleType: "退货退款",
		CardKeys:      []model.AftersaleTicketCard{{CardKey: "待商家收/发货:退货待收货"}},
	}
	if !MatchShopTicketKind(returnWaitReceive, dto.TicketKindReturnRefund) {
		t.Fatal("退货待收货 should match 退货退款")
	}
	if MatchShopTicketKind(returnWaitReceive, dto.TicketKindExchange) {
		t.Fatal("退货待收货 should not match 换货")
	}
	if MatchShopTicketKind(review, dto.TicketKindReturnRefund) {
		t.Fatal("已发货退款 should not match 退货退款")
	}

	buyerPendingReturn := &model.AftersaleTicket{
		CardKeys: []model.AftersaleTicketCard{{CardKey: "待消费者处理:待退货"}},
	}
	if !MatchShopTicketKind(buyerPendingReturn, dto.TicketKindReturnRefund) {
		t.Fatal("待消费者处理 待退货 should match 退货退款")
	}

	exchangeReceive := &model.AftersaleTicket{
		AftersaleType: "换货",
		CardKeys:      []model.AftersaleTicketCard{{CardKey: "待商家收/发货:换货待收货/发货"}},
	}
	if !MatchShopTicketKind(exchangeReceive, dto.TicketKindExchange) {
		t.Fatal("换货待收货/发货 should match 换货")
	}
	if MatchShopTicketKind(exchangeReceive, dto.TicketKindReturnRefund) {
		t.Fatal("换货 should not match 退货退款")
	}

	exchangeByType := &model.AftersaleTicket{
		AftersaleType: "换货",
		CardKeys:      []model.AftersaleTicketCard{{CardKey: "紧急:临期待处理"}},
	}
	if !MatchShopTicketKind(exchangeByType, dto.TicketKindExchange) {
		t.Fatal("aftersale type 换货 should match even off the receive card")
	}
	disputeNegotiate := &model.AftersaleTicket{
		CardKeys: []model.AftersaleTicketCard{{CardKey: "纠纷:仲裁待协商"}},
	}
	if !MatchShopTicketKind(disputeNegotiate, dto.TicketKindDispute) {
		t.Fatal("仲裁待协商 should match dispute")
	}
}

func TestInterceptFromTicketCopiesTimeout(t *testing.T) {
	deadline := time.Date(2026, 9, 26, 22, 11, 18, 0, time.Local)
	item := interceptFromTicket(&model.AftersaleTicket{
		Status:        "待商家处理",
		TimeoutText:   "5天3小时41分后自动同意",
		TimeoutAction: "自动同意",
		DeadlineAt:    &deadline,
		Logistics:     "订单发货 待取件\n需商家拦截快递",
	}, "测试店")
	if item.Status != "待商家处理" {
		t.Fatalf("status=%q", item.Status)
	}
	if item.TimeoutText != "5天3小时41分后自动同意" {
		t.Fatalf("timeoutText=%q", item.TimeoutText)
	}
	if item.TimeoutDisplay == "" || item.DeadlineAt == "" {
		t.Fatalf("timeoutDisplay=%q deadlineAt=%q", item.TimeoutDisplay, item.DeadlineAt)
	}
}

func TestShopTicketKeywordMatchPickupPoint(t *testing.T) {
	ticket := &model.AftersaleTicket{
		PlatformAftersaleID: "AS1",
		OrderNo:             "O1",
		TrackJSON: `[
			{"date":"09/12 11:37:01","title":"待取件","detail":"快件已存放至菜鸟-杭州萧山利二花苑店，请及时取件"},
			{"date":"09/12 07:23:57","title":"派件中","detail":"正在派件"}
		]`,
	}
	cases := []string{"利二花苑", "萧山 利二", "菜鸟杭州", "花苑店"}
	for _, kw := range cases {
		if !shopTicketKeywordMatch(ticket, "", kw) {
			t.Fatalf("keyword %q should fuzzy-match pickup point", kw)
		}
	}
	if shopTicketKeywordMatch(ticket, "", "不存在驿站") {
		t.Fatal("unrelated keyword should not match")
	}
}
