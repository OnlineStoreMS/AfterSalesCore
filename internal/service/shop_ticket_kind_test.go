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
}
