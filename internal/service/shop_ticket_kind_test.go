package service

import (
	"testing"

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
	if MatchShopTicketKind(signedOtherCard, dto.TicketKindBuyerReturnSigned) {
		t.Fatal("signed without 全部待收货/发货 should not match")
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
}
