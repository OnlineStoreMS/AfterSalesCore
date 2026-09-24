package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"aftersalescore/internal/dto"
	"aftersalescore/internal/integrations/ordercore"
	"aftersalescore/internal/model"
	"aftersalescore/internal/repo"

	"gorm.io/gorm"
)

var issueProblemTypes = []dto.IssueOption{
	{Value: "wrong_item", Label: "发错货"},
	{Value: "missing_item", Label: "少发/漏发"},
	{Value: "extra_item", Label: "多发"},
	{Value: "wrong_spec", Label: "错发颜色/尺码"},
	{Value: "damaged", Label: "破损"},
	{Value: "quality", Label: "质量问题"},
	{Value: "exchange", Label: "换货诉求"},
	{Value: "not_received", Label: "未收到货"},
	{Value: "logistics", Label: "快递异常"},
	{Value: "gift", Label: "赠品问题"},
	{Value: "invoice", Label: "发票/票据"},
	{Value: model.IssueProblemCustom, Label: "自定义"},
}

var issueHandleMethods = []dto.IssueOption{
	{Value: "reship", Label: "补发"},
	{Value: "exchange", Label: "退换"},
	{Value: "partial_refund", Label: "部分退款"},
	{Value: "full_refund", Label: "全额退款"},
	{Value: "negotiate_return", Label: "协商退货退款"},
	{Value: "compensate", Label: "仅补偿（优惠券/红包）"},
	{Value: "intercept", Label: "拦截改址"},
	{Value: "reship_parts", Label: "补发配件"},
	{Value: "close", Label: "协商关闭"},
	{Value: model.IssueHandleCustom, Label: "自定义"},
}

var issueStatuses = []dto.IssueOption{
	{Value: model.IssueStatusPending, Label: "待处理"},
	{Value: model.IssueStatusProcessing, Label: "处理中"},
	{Value: model.IssueStatusDone, Label: "已完成"},
}

func (s *ShopService) IssueMeta() dto.IssueMeta {
	return dto.IssueMeta{
		ProblemTypes:  append([]dto.IssueOption(nil), issueProblemTypes...),
		HandleMethods: append([]dto.IssueOption(nil), issueHandleMethods...),
		Statuses:      append([]dto.IssueOption(nil), issueStatuses...),
	}
}

func (s *ShopService) issueRepo() *repo.IssueRepo {
	return s.repos.Issue.ForTenant(s.tenantID)
}

func (s *ShopService) LookupIssueSource(keyword, bearerToken string) ([]dto.IssueLookupResult, error) {
	kw := strings.TrimSpace(keyword)
	if kw == "" {
		return nil, fmt.Errorf("%w: 请填写订单号或售后单号", ErrBadRequest)
	}

	tickets, err := s.repo().FindTicketsByKeyword(kw, 20)
	if err != nil {
		return nil, err
	}
	shops, err := s.repo().List()
	if err != nil {
		return nil, err
	}
	shopNames := map[uint64]string{}
	for i := range shops {
		shopNames[shops[i].ID] = shops[i].Name
	}

	orderNos := []string{kw}
	for i := range tickets {
		if n := strings.TrimSpace(tickets[i].OrderNo); n != "" {
			orderNos = append(orderNos, n)
		}
	}
	orderMap := map[string]ordercore.OrderSummary{}
	if s.orders != nil && strings.TrimSpace(bearerToken) != "" {
		summaries, err := s.orders.LookupSummaries(context.Background(), bearerToken, orderNos)
		if err != nil {
			log.Printf("[aftersales] 问题单检索订单失败: %v", err)
		} else {
			orderMap = summaries
		}
	}

	out := make([]dto.IssueLookupResult, 0, len(tickets)+1)
	seen := map[string]struct{}{}
	for i := range tickets {
		t := &tickets[i]
		key := strings.TrimSpace(t.PlatformAftersaleID)
		if key == "" {
			key = fmt.Sprintf("t-%d", t.ID)
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		item := dto.IssueLookupResult{
			OrderNo:             strings.TrimSpace(t.OrderNo),
			PlatformAftersaleID: strings.TrimSpace(t.PlatformAftersaleID),
			ShopID:              t.ShopID,
			ShopName:            shopNames[t.ShopID],
			ProductTitle:        strings.TrimSpace(t.ProductTitle),
			ProductImage:        strings.TrimSpace(t.ProductImage),
			SkuSpecs:            strings.TrimSpace(t.SKU),
			HasAftersale:        true,
			AftersaleStatus:     strings.TrimSpace(t.Status),
			AftersaleType:       strings.TrimSpace(t.AftersaleType),
			Logistics:           strings.TrimSpace(t.Logistics),
			ReturnLogisticsNo:   strings.TrimSpace(t.ReturnLogisticsNo),
			ShipLogisticsNo:     strings.TrimSpace(t.ShipLogisticsNo),
			Source:              "ticket",
		}
		if ord, ok := orderMap[item.OrderNo]; ok {
			mergeLookupFromOrder(&item, ord)
			item.Source = "both"
		} else if ord, ok := orderMap[kw]; ok && item.OrderNo == "" {
			mergeLookupFromOrder(&item, ord)
			item.Source = "both"
		}
		out = append(out, item)
	}

	if len(out) == 0 {
		if ord, ok := orderMap[kw]; ok {
			item := dto.IssueLookupResult{Source: "order", HasAftersale: false}
			mergeLookupFromOrder(&item, ord)
			out = append(out, item)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: 未找到订单或售后单", ErrNotFound)
	}
	return out, nil
}

func mergeLookupFromOrder(item *dto.IssueLookupResult, ord ordercore.OrderSummary) {
	if item.OrderNo == "" {
		item.OrderNo = ord.OrderNo
	}
	if item.PlatformOrderID == "" {
		item.PlatformOrderID = ord.PlatformOrderID
	}
	if item.ShopName == "" {
		item.ShopName = ord.ShopName
	}
	if item.ProductTitle == "" {
		item.ProductTitle = ord.ProductTitle
	}
	if item.ProductImage == "" {
		item.ProductImage = ord.ProductImage
	}
	if item.SkuSpecs == "" {
		item.SkuSpecs = ord.SkuSpecs
	}
	if item.BuyerName == "" {
		item.BuyerName = ord.BuyerName
	}
	if item.BuyerPhone == "" {
		item.BuyerPhone = ord.BuyerPhone
	}
	if item.BuyerAddress == "" {
		item.BuyerAddress = ord.Address
	}
}

func (s *ShopService) ListIssues(q dto.IssueListQuery) ([]dto.IssueItem, int64, error) {
	list, total, err := s.issueRepo().List(repo.IssueListFilter{
		ShopID: q.ShopID, Status: q.Status, ProblemType: q.ProblemType,
		Keyword: q.Keyword, Page: q.Page, PageSize: q.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}
	out := make([]dto.IssueItem, 0, len(list))
	for i := range list {
		out = append(out, toIssueItem(&list[i]))
	}
	return out, total, nil
}

func (s *ShopService) GetIssue(id uint64) (*dto.IssueItem, error) {
	row, err := s.issueRepo().Get(id)
	if err != nil {
		if errorsIsNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	item := toIssueItem(row)
	return &item, nil
}

func (s *ShopService) CreateIssue(in dto.IssueUpsertRequest, operatorID uint64, operatorName string) (*dto.IssueItem, error) {
	row, err := buildIssueRow(in, operatorID, operatorName)
	if err != nil {
		return nil, err
	}
	row.Status = normalizeIssueStatus(in.Status)
	if row.Status == "" {
		row.Status = model.IssueStatusPending
	}
	if err := s.issueRepo().Create(row); err != nil {
		return nil, err
	}
	item := toIssueItem(row)
	return &item, nil
}

func (s *ShopService) UpdateIssue(id uint64, in dto.IssueUpsertRequest, operatorID uint64, operatorName string) (*dto.IssueItem, error) {
	existing, err := s.issueRepo().Get(id)
	if err != nil {
		if errorsIsNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	row, err := buildIssueRow(in, operatorID, operatorName)
	if err != nil {
		return nil, err
	}
	row.ID = existing.ID
	row.TenantID = existing.TenantID
	row.CreatedAt = existing.CreatedAt
	row.Status = normalizeIssueStatus(in.Status)
	if row.Status == "" {
		row.Status = existing.Status
	}
	if row.Status == model.IssueStatusDone {
		now := time.Now()
		if existing.CompletedAt != nil {
			row.CompletedAt = existing.CompletedAt
		} else {
			row.CompletedAt = &now
		}
	} else {
		row.CompletedAt = nil
	}
	if err := s.issueRepo().Update(row); err != nil {
		return nil, err
	}
	item := toIssueItem(row)
	return &item, nil
}

func (s *ShopService) UpdateIssueStatus(id uint64, status string, operatorID uint64, operatorName string) (*dto.IssueItem, error) {
	row, err := s.issueRepo().Get(id)
	if err != nil {
		if errorsIsNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	st := normalizeIssueStatus(status)
	if st == "" {
		return nil, fmt.Errorf("%w: 无效状态", ErrBadRequest)
	}
	row.Status = st
	if operatorName != "" {
		row.OperatorName = operatorName
		row.OperatorID = operatorID
	}
	if st == model.IssueStatusDone {
		now := time.Now()
		row.CompletedAt = &now
	} else {
		row.CompletedAt = nil
	}
	if err := s.issueRepo().Update(row); err != nil {
		return nil, err
	}
	item := toIssueItem(row)
	return &item, nil
}

func (s *ShopService) DeleteIssue(id uint64) error {
	if err := s.issueRepo().Delete(id); err != nil {
		return err
	}
	return nil
}

func buildIssueRow(in dto.IssueUpsertRequest, operatorID uint64, operatorName string) (*model.AftersaleIssue, error) {
	shopName := strings.TrimSpace(in.ShopName)
	product := strings.TrimSpace(in.ProductTitle)
	orderNo := strings.TrimSpace(in.OrderNo)
	aftersaleID := strings.TrimSpace(in.PlatformAftersaleID)
	if shopName == "" {
		return nil, fmt.Errorf("%w: 店铺必填", ErrBadRequest)
	}
	if product == "" {
		return nil, fmt.Errorf("%w: 商品信息必填", ErrBadRequest)
	}
	if orderNo == "" && aftersaleID == "" {
		return nil, fmt.Errorf("%w: 订单号或售后单号至少填一项", ErrBadRequest)
	}
	problemType := strings.TrimSpace(in.ProblemType)
	if problemType == "" {
		return nil, fmt.Errorf("%w: 请选择问题类型", ErrBadRequest)
	}
	problemCustom := strings.TrimSpace(in.ProblemTypeCustom)
	if problemType == model.IssueProblemCustom && problemCustom == "" {
		return nil, fmt.Errorf("%w: 请填写自定义问题类型", ErrBadRequest)
	}
	handleMethod := strings.TrimSpace(in.HandleMethod)
	handleCustom := strings.TrimSpace(in.HandleMethodCustom)
	if handleMethod == model.IssueHandleCustom && handleCustom == "" {
		return nil, fmt.Errorf("%w: 请填写自定义处理方式", ErrBadRequest)
	}
	hasAftersale := in.HasAftersale || aftersaleID != ""
	return &model.AftersaleIssue{
		ShopID:              in.ShopID,
		ShopName:            shopName,
		OrderNo:             orderNo,
		PlatformOrderID:     strings.TrimSpace(in.PlatformOrderID),
		PlatformAftersaleID: aftersaleID,
		ProductTitle:        product,
		ProductImage:        strings.TrimSpace(in.ProductImage),
		SkuSpecs:            strings.TrimSpace(in.SkuSpecs),
		BuyerName:           strings.TrimSpace(in.BuyerName),
		BuyerPhone:          strings.TrimSpace(in.BuyerPhone),
		BuyerAddress:        strings.TrimSpace(in.BuyerAddress),
		HasAftersale:        hasAftersale,
		AftersaleStatus:     strings.TrimSpace(in.AftersaleStatus),
		AftersaleType:       strings.TrimSpace(in.AftersaleType),
		Logistics:           strings.TrimSpace(in.Logistics),
		ReturnLogisticsNo:   strings.TrimSpace(in.ReturnLogisticsNo),
		ShipLogisticsNo:     strings.TrimSpace(in.ShipLogisticsNo),
		ProblemType:         problemType,
		ProblemTypeCustom:   problemCustom,
		HandleMethod:        handleMethod,
		HandleMethodCustom:  handleCustom,
		ProblemNote:         strings.TrimSpace(in.ProblemNote),
		HandleNote:          strings.TrimSpace(in.HandleNote),
		OperatorID:          operatorID,
		OperatorName:        strings.TrimSpace(operatorName),
	}, nil
}

func normalizeIssueStatus(v string) string {
	switch strings.TrimSpace(v) {
	case model.IssueStatusPending, "待处理":
		return model.IssueStatusPending
	case model.IssueStatusProcessing, "处理中":
		return model.IssueStatusProcessing
	case model.IssueStatusDone, "已完成":
		return model.IssueStatusDone
	default:
		return ""
	}
}

func issueOptionLabel(opts []dto.IssueOption, value, custom string) string {
	value = strings.TrimSpace(value)
	custom = strings.TrimSpace(custom)
	if value == model.IssueProblemCustom || value == model.IssueHandleCustom {
		if custom != "" {
			return custom
		}
		return "自定义"
	}
	for _, o := range opts {
		if o.Value == value {
			return o.Label
		}
	}
	if custom != "" {
		return custom
	}
	return value
}

func toIssueItem(row *model.AftersaleIssue) dto.IssueItem {
	item := dto.IssueItem{
		ID: row.ID, ShopID: row.ShopID, ShopName: row.ShopName,
		OrderNo: row.OrderNo, PlatformOrderID: row.PlatformOrderID, PlatformAftersaleID: row.PlatformAftersaleID,
		ProductTitle: row.ProductTitle, ProductImage: row.ProductImage, SkuSpecs: row.SkuSpecs,
		BuyerName: row.BuyerName, BuyerPhone: row.BuyerPhone, BuyerAddress: row.BuyerAddress,
		HasAftersale: row.HasAftersale, AftersaleStatus: row.AftersaleStatus, AftersaleType: row.AftersaleType,
		Logistics: row.Logistics, ReturnLogisticsNo: row.ReturnLogisticsNo, ShipLogisticsNo: row.ShipLogisticsNo,
		ProblemType: row.ProblemType, ProblemTypeCustom: row.ProblemTypeCustom,
		ProblemTypeLabel: issueOptionLabel(issueProblemTypes, row.ProblemType, row.ProblemTypeCustom),
		HandleMethod: row.HandleMethod, HandleMethodCustom: row.HandleMethodCustom,
		HandleMethodLabel: issueOptionLabel(issueHandleMethods, row.HandleMethod, row.HandleMethodCustom),
		ProblemNote: row.ProblemNote, HandleNote: row.HandleNote,
		Status: row.Status, StatusLabel: issueOptionLabel(issueStatuses, row.Status, ""),
		OperatorName: row.OperatorName,
		CreatedAt:    formatTime(row.CreatedAt),
		UpdatedAt:    formatTime(row.UpdatedAt),
	}
	if row.CompletedAt != nil {
		t := formatTime(*row.CompletedAt)
		item.CompletedAt = &t
	}
	return item
}

func errorsIsNotFound(err error) bool {
	return err == gorm.ErrRecordNotFound
}
