package repo

import (
	"strings"
	"time"

	"aftersalescore/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ShopRepo struct {
	db       *gorm.DB
	tenantID uint64
}

func NewShopRepo(db *gorm.DB) *ShopRepo {
	return &ShopRepo{db: db}
}

func (r *ShopRepo) ForTenant(tenantID uint64) *ShopRepo {
	return &ShopRepo{db: r.db, tenantID: NormalizeTenantID(tenantID)}
}

func (r *ShopRepo) DB() *gorm.DB {
	return r.db
}

func (r *ShopRepo) List() ([]model.MarketplaceShop, error) {
	var list []model.MarketplaceShop
	err := r.db.Scopes(scopeTenant(r.tenantID)).Order("id DESC").Find(&list).Error
	return list, err
}

func (r *ShopRepo) CountPendingTickets() (map[uint64]int, error) {
	type row struct {
		ShopID uint64
		Count  int
	}
	var rows []row
	err := r.db.Model(&model.AftersaleTicket{}).
		Scopes(scopeTenant(r.tenantID)).
		Select("shop_id, count(*) as count").
		Where("id IN (?)", r.db.Model(&model.AftersaleTicketCard{}).Select("ticket_id")).
		Group("shop_id").
		Scan(&rows).Error
	out := make(map[uint64]int, len(rows))
	for _, x := range rows {
		out[x.ShopID] = x.Count
	}
	return out, err
}

func (r *ShopRepo) Get(id uint64) (*model.MarketplaceShop, error) {
	var shop model.MarketplaceShop
	err := r.db.Scopes(scopeTenant(r.tenantID)).First(&shop, id).Error
	if err != nil {
		return nil, err
	}
	return &shop, nil
}

func (r *ShopRepo) GetByBindCode(code string) (*model.MarketplaceShop, error) {
	var shop model.MarketplaceShop
	err := r.db.Where("bind_code = ?", strings.TrimSpace(code)).First(&shop).Error
	if err != nil {
		return nil, err
	}
	return &shop, nil
}

func (r *ShopRepo) GetByPluginKey(key string) (*model.MarketplaceShop, error) {
	var shop model.MarketplaceShop
	err := r.db.Where("plugin_key = ?", strings.TrimSpace(key)).First(&shop).Error
	if err != nil {
		return nil, err
	}
	return &shop, nil
}

func (r *ShopRepo) Create(shop *model.MarketplaceShop) error {
	shop.TenantID = r.tenantID
	return r.db.Create(shop).Error
}

func (r *ShopRepo) Save(shop *model.MarketplaceShop) error {
	return r.db.Save(shop).Error
}

func (r *ShopRepo) Delete(id uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var tickets []model.AftersaleTicket
		if err := tx.Where("shop_id = ? AND tenant_id = ?", id, r.tenantID).Find(&tickets).Error; err != nil {
			return err
		}
		ids := make([]uint64, 0, len(tickets))
		for _, t := range tickets {
			ids = append(ids, t.ID)
		}
		if len(ids) > 0 {
			if err := tx.Where("ticket_id IN ?", ids).Delete(&model.AftersaleTicketCard{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("shop_id = ? AND tenant_id = ?", id, r.tenantID).Delete(&model.AftersaleTicket{}).Error; err != nil {
			return err
		}
		if err := tx.Where("shop_id = ? AND tenant_id = ?", id, r.tenantID).Delete(&model.AftersaleFilterCard{}).Error; err != nil {
			return err
		}
		if err := tx.Where("shop_id = ? AND tenant_id = ?", id, r.tenantID).Delete(&model.ServiceOrder{}).Error; err != nil {
			return err
		}
		if err := tx.Where("shop_id = ? AND tenant_id = ?", id, r.tenantID).Delete(&model.ReturnPackage{}).Error; err != nil {
			return err
		}
		if err := tx.Where("shop_id = ? AND tenant_id = ?", id, r.tenantID).Delete(&model.ShippedRefundSuccess{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ? AND tenant_id = ?", id, r.tenantID).Delete(&model.MarketplaceShop{}).Error
	})
}

func (r *ShopRepo) ReplaceCards(shop *model.MarketplaceShop, cards []model.AftersaleFilterCard) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("shop_id = ?", shop.ID).Delete(&model.AftersaleFilterCard{}).Error; err != nil {
			return err
		}
		if len(cards) == 0 {
			return nil
		}
		return tx.Create(&cards).Error
	})
}

func (r *ShopRepo) ListCards(shopID uint64) ([]model.AftersaleFilterCard, error) {
	var list []model.AftersaleFilterCard
	err := r.db.Where("shop_id = ? AND tenant_id = ?", shopID, r.tenantID).
		Order("sort_order ASC, id ASC").Find(&list).Error
	return list, err
}

type TicketListFilter struct {
	ShopID   uint64
	CardKey  string
	Keyword  string
	Page     int
	PageSize int
}

func (r *ShopRepo) ListTickets(f TicketListFilter) ([]model.AftersaleTicket, int64, error) {
	q := r.db.Model(&model.AftersaleTicket{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("shop_id = ?", f.ShopID)
	if f.CardKey != "" {
		q = q.Where("id IN (?)",
			r.db.Model(&model.AftersaleTicketCard{}).Select("ticket_id").Where("card_key = ?", f.CardKey),
		)
	} else {
		q = q.Where("id IN (?)",
			r.db.Model(&model.AftersaleTicketCard{}).Select("ticket_id"),
		)
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where(
			"platform_aftersale_id ILIKE ? OR order_no ILIKE ? OR product_title ILIKE ? OR status ILIKE ? OR return_logistics_no ILIKE ? OR ship_logistics_no ILIKE ?",
			like, like, like, like, like, like,
		)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 200 {
		f.PageSize = 20
	}
	var list []model.AftersaleTicket
	offset := (f.Page - 1) * f.PageSize
	err := q.Preload("CardKeys").
		Order("deadline_at ASC NULLS LAST, id DESC").
		Offset(offset).Limit(f.PageSize).Find(&list).Error
	return list, total, err
}

func (r *ShopRepo) UpsertTickets(shop *model.MarketplaceShop, tickets []model.AftersaleTicket, cardKeys map[string][]string) error {
	now := time.Now()
	return r.db.Transaction(func(tx *gorm.DB) error {
		seenIDs := make([]uint64, 0, len(tickets))
		for i := range tickets {
			t := &tickets[i]
			t.TenantID = shop.TenantID
			t.ShopID = shop.ID
			t.SyncedAt = now
			var existing model.AftersaleTicket
			err := tx.Where("shop_id = ? AND platform_aftersale_id = ?", shop.ID, t.PlatformAftersaleID).
				First(&existing).Error
			if err == gorm.ErrRecordNotFound {
				if err := tx.Create(t).Error; err != nil {
					return err
				}
			} else if err != nil {
				return err
			} else {
				t.ID = existing.ID
				t.CreatedAt = existing.CreatedAt
				var deadline any
				if t.DeadlineAt != nil {
					deadline = *t.DeadlineAt
				}
				updates := map[string]any{
					"order_no":       t.OrderNo,
					"product_title":  t.ProductTitle,
					"product_image":  t.ProductImage,
					"sku":            t.SKU,
					"product_tags":   t.ProductTags,
					"tags":           t.Tags,
					"qty":            t.Qty,
					"buy_qty":        t.BuyQty,
					"pay_amount":     t.PayAmount,
					"refund_amount":  t.RefundAmount,
					"aftersale_type": t.AftersaleType,
					"reason":         t.Reason,
					"status":         t.Status,
					"timeout_text":   t.TimeoutText,
					"timeout_action": t.TimeoutAction,
					"deadline_at":    deadline,
					"dispute":        t.Dispute,
					"logistics":      t.Logistics,
					"apply_time":     t.ApplyTime,
					"raw_json":       t.RawJSON,
					"synced_at":      now,
				}
				if t.ReturnLogisticsNo != "" {
					updates["return_logistics_no"] = t.ReturnLogisticsNo
				}
				if t.ShipLogisticsNo != "" {
					updates["ship_logistics_no"] = t.ShipLogisticsNo
				}
				if err := tx.Model(&existing).Updates(updates).Error; err != nil {
					return err
				}
			}
			if err := tx.Where("ticket_id = ?", t.ID).Delete(&model.AftersaleTicketCard{}).Error; err != nil {
				return err
			}
			keys := uniqueStrings(cardKeys[t.PlatformAftersaleID])
			seenIDs = append(seenIDs, t.ID)
			if len(keys) == 0 {
				continue
			}
			rels := make([]model.AftersaleTicketCard, 0, len(keys))
			for _, k := range keys {
				rels = append(rels, model.AftersaleTicketCard{TicketID: t.ID, CardKey: k})
			}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&rels).Error; err != nil {
				return err
			}
		}
		q := tx.Where("ticket_id IN (?)", tx.Model(&model.AftersaleTicket{}).Select("id").Where("shop_id = ?", shop.ID))
		if len(seenIDs) > 0 {
			q = q.Where("ticket_id NOT IN ?", seenIDs)
		}
		if err := q.Delete(&model.AftersaleTicketCard{}).Error; err != nil {
			return err
		}
		left := tx.Where("shop_id = ?", shop.ID)
		if len(seenIDs) > 0 {
			left = left.Where("id NOT IN ?", seenIDs)
		}
		return left.Delete(&model.AftersaleTicket{}).Error
	})
}

func uniqueStrings(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func (r *ShopRepo) ListCardsForTenant() ([]model.AftersaleFilterCard, error) {
	var list []model.AftersaleFilterCard
	err := r.db.Scopes(scopeTenant(r.tenantID)).Order("sort_order ASC, id ASC").Find(&list).Error
	return list, err
}

func (r *ShopRepo) ListTicketsByCard(shopID uint64, cardKey string) ([]model.AftersaleTicket, error) {
	var list []model.AftersaleTicket
	q := r.db.Scopes(scopeTenant(r.tenantID)).Where("shop_id = ?", shopID)
	if cardKey != "" {
		q = q.Where("id IN (?)",
			r.db.Model(&model.AftersaleTicketCard{}).Select("ticket_id").Where("card_key = ?", cardKey),
		)
	}
	err := q.Limit(200).Find(&list).Error
	return list, err
}

func (r *ShopRepo) ListTicketsWithDeadline(shopID uint64) ([]model.AftersaleTicket, error) {
	var list []model.AftersaleTicket
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Where("shop_id = ? AND deadline_at IS NOT NULL", shopID).
		Limit(200).Find(&list).Error
	return list, err
}

type ReturnListFilter struct {
	ShopID     uint64
	Keyword    string
	ReturnFrom *time.Time
	ReturnTo   *time.Time
	ApplyFrom  *time.Time
	ApplyTo    *time.Time
	Page       int
	PageSize   int
}

func (r *ShopRepo) ListReturns(f ReturnListFilter) ([]model.ReturnPackage, int64, error) {
	q := r.db.Model(&model.ReturnPackage{}).Scopes(scopeTenant(r.tenantID))
	if f.ShopID > 0 {
		q = q.Where("shop_id = ?", f.ShopID)
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where(
			"platform_aftersale_id ILIKE ? OR order_no ILIKE ? OR product_title ILIKE ? OR sku ILIKE ? OR logistics_no ILIKE ? OR return_location ILIKE ? OR carrier ILIKE ? OR status ILIKE ? OR order_info ILIKE ? OR aftersale_info ILIKE ?",
			like, like, like, like, like, like, like, like, like, like,
		)
	}
	if f.ReturnFrom != nil {
		q = q.Where("returned_at >= ?", f.ReturnFrom)
	}
	if f.ReturnTo != nil {
		q = q.Where("returned_at <= ?", f.ReturnTo)
	}
	if f.ApplyFrom != nil {
		q = q.Where("applied_at >= ?", f.ApplyFrom)
	}
	if f.ApplyTo != nil {
		q = q.Where("applied_at <= ?", f.ApplyTo)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 200 {
		f.PageSize = 20
	}
	var list []model.ReturnPackage
	offset := (f.Page - 1) * f.PageSize
	err := q.Order("COALESCE(returned_at, applied_at) DESC NULLS LAST, id DESC").
		Offset(offset).Limit(f.PageSize).Find(&list).Error
	return list, total, err
}

func (r *ShopRepo) UpsertReturns(shop *model.MarketplaceShop, items []model.ReturnPackage) error {
	if len(items) == 0 {
		return nil
	}
	now := time.Now()
	return r.db.Transaction(func(tx *gorm.DB) error {
		aftersaleIDs := make([]string, 0, len(items))
		for i := range items {
			item := &items[i]
			item.TenantID = shop.TenantID
			item.ShopID = shop.ID
			item.SyncedAt = now
			if item.PlatformAftersaleID != "" {
				aftersaleIDs = append(aftersaleIDs, item.PlatformAftersaleID)
			}
			var existing model.ReturnPackage
			err := tx.Where("shop_id = ? AND platform_aftersale_id = ?", shop.ID, item.PlatformAftersaleID).
				First(&existing).Error
			if err == gorm.ErrRecordNotFound {
				if err := tx.Create(item).Error; err != nil {
					return err
				}
				continue
			}
			if err != nil {
				return err
			}
			updates := map[string]any{
				"order_no":       item.OrderNo,
				"product_title":  item.ProductTitle,
				"product_image":  item.ProductImage,
				"sku":            item.SKU,
				"qty":            item.Qty,
				"buy_qty":        item.BuyQty,
				"pay_amount":     item.PayAmount,
				"refund_amount":  item.RefundAmount,
				"aftersale_type": item.AftersaleType,
				"reason":         item.Reason,
				"status":         item.Status,
				"order_info":     item.OrderInfo,
				"aftersale_info": item.AftersaleInfo,
				"logistics":      item.Logistics,
				"apply_time":     item.ApplyTime,
				"applied_at":     item.AppliedAt,
				"raw_json":       item.RawJSON,
				"synced_at":      now,
			}
			if item.LogisticsNo != "" {
				updates["logistics_no"] = item.LogisticsNo
			}
			if item.Carrier != "" {
				updates["carrier"] = item.Carrier
			}
			if item.ReturnLocation != "" {
				updates["return_location"] = item.ReturnLocation
			}
			if item.ShipTime != "" {
				updates["ship_time"] = item.ShipTime
			}
			if item.ReturnTime != "" {
				updates["return_time"] = item.ReturnTime
			}
			if item.ReturnedAt != nil {
				updates["returned_at"] = item.ReturnedAt
			}
			if item.TrackJSON != "" {
				updates["track_json"] = item.TrackJSON
			}
			if err := tx.Model(&existing).Updates(updates).Error; err != nil {
				return err
			}
		}
		if len(aftersaleIDs) > 0 {
			if err := tx.Where("shop_id = ? AND platform_aftersale_id IN ?", shop.ID, aftersaleIDs).
				Delete(&model.ShippedRefundSuccess{}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

type ShippedRefundListFilter struct {
	ShopID    uint64
	Keyword   string
	Status    string
	AlertOnly bool
	ApplyFrom *time.Time
	ApplyTo   *time.Time
	Page      int
	PageSize  int
	Unpaged   bool
}

func (r *ShopRepo) ListShippedRefunds(f ShippedRefundListFilter) ([]model.ShippedRefundSuccess, int64, error) {
	q := r.db.Model(&model.ShippedRefundSuccess{}).Scopes(scopeTenant(r.tenantID))
	if f.ShopID > 0 {
		q = q.Where("shop_id = ?", f.ShopID)
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where(
			"platform_aftersale_id ILIKE ? OR order_no ILIKE ? OR product_title ILIKE ? OR sku ILIKE ? OR logistics ILIKE ? OR order_info ILIKE ? OR aftersale_info ILIKE ? OR status ILIKE ? OR track_json ILIKE ? OR logistics_no ILIKE ?",
			like, like, like, like, like, like, like, like, like, like,
		)
	}
	if status := strings.TrimSpace(f.Status); status != "" {
		q = q.Where("logistics_status = ?", status)
	}
	if f.AlertOnly {
		q = q.Where("logistics_status IN ?", []string{"待取件", "已签收", "运输中"})
	}
	if f.ApplyFrom != nil {
		q = q.Where("applied_at >= ?", f.ApplyFrom)
	}
	if f.ApplyTo != nil {
		q = q.Where("applied_at <= ?", f.ApplyTo)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	q = q.Order("COALESCE(applied_at, synced_at) DESC NULLS LAST, id DESC")
	var list []model.ShippedRefundSuccess
	if f.Unpaged {
		err := q.Find(&list).Error
		return list, total, err
	}
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 200 {
		f.PageSize = 20
	}
	offset := (f.Page - 1) * f.PageSize
	err := q.Offset(offset).Limit(f.PageSize).Find(&list).Error
	return list, total, err
}

func (r *ShopRepo) ListInterceptTickets(shopID uint64, keyword string) ([]model.AftersaleTicket, error) {
	q := r.db.Model(&model.AftersaleTicket{}).Scopes(scopeTenant(r.tenantID)).
		Where("logistics LIKE ?", "%需商家拦截快递%")
	if shopID > 0 {
		q = q.Where("shop_id = ?", shopID)
	}
	if kw := strings.TrimSpace(keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where(
			"platform_aftersale_id ILIKE ? OR order_no ILIKE ? OR product_title ILIKE ? OR sku ILIKE ? OR logistics ILIKE ? OR ship_logistics_no ILIKE ? OR return_logistics_no ILIKE ?",
			like, like, like, like, like, like, like,
		)
	}
	var list []model.AftersaleTicket
	err := q.Order("synced_at DESC, id DESC").Limit(500).Find(&list).Error
	return list, err
}

func (r *ShopRepo) ListShippedRefundsByStatus(shopID uint64, status string) ([]model.ShippedRefundSuccess, error) {
	var list []model.ShippedRefundSuccess
	q := r.db.Model(&model.ShippedRefundSuccess{}).Scopes(scopeTenant(r.tenantID)).
		Where("logistics_status = ?", status)
	if shopID > 0 {
		q = q.Where("shop_id = ?", shopID)
	}
	err := q.Order("COALESCE(applied_at, synced_at) DESC").Find(&list).Error
	return list, err
}

func (r *ShopRepo) UpsertShippedRefunds(shop *model.MarketplaceShop, items []model.ShippedRefundSuccess) error {
	if len(items) == 0 {
		return nil
	}
	now := time.Now()
	return r.db.Transaction(func(tx *gorm.DB) error {
		for i := range items {
			item := &items[i]
			item.TenantID = shop.TenantID
			item.ShopID = shop.ID
			item.SyncedAt = now
			var existing model.ShippedRefundSuccess
			err := tx.Where("shop_id = ? AND platform_aftersale_id = ?", shop.ID, item.PlatformAftersaleID).
				First(&existing).Error
			if err == gorm.ErrRecordNotFound {
				if err := tx.Create(item).Error; err != nil {
					return err
				}
				continue
			}
			if err != nil {
				return err
			}
			updates := map[string]any{
				"order_no":         item.OrderNo,
				"product_title":    item.ProductTitle,
				"product_image":    item.ProductImage,
				"sku":              item.SKU,
				"product_tags":     item.ProductTags,
				"tags":             item.Tags,
				"qty":              item.Qty,
				"buy_qty":          item.BuyQty,
				"pay_amount":       item.PayAmount,
				"refund_amount":    item.RefundAmount,
				"aftersale_type":   item.AftersaleType,
				"reason":           item.Reason,
				"status":           item.Status,
				"order_info":       item.OrderInfo,
				"aftersale_info":   item.AftersaleInfo,
				"logistics":        item.Logistics,
				"logistics_status": item.LogisticsStatus,
				"apply_time":       item.ApplyTime,
				"applied_at":       item.AppliedAt,
				"raw_json":         item.RawJSON,
				"synced_at":        now,
			}
			if item.LogisticsNo != "" {
				updates["logistics_no"] = item.LogisticsNo
			}
			if item.Carrier != "" {
				updates["carrier"] = item.Carrier
			}
			if item.ShipTime != "" {
				updates["ship_time"] = item.ShipTime
			}
			if item.TrackJSON != "" {
				updates["track_json"] = item.TrackJSON
			}
			if err := tx.Model(&existing).Updates(updates).Error; err != nil {
				return err
			}
		}
		ids := make([]string, 0, len(items))
		for i := range items {
			if id := strings.TrimSpace(items[i].PlatformAftersaleID); id != "" {
				ids = append(ids, id)
			}
		}
		if len(ids) == 0 {
			return nil
		}
		return tx.Where("shop_id = ? AND tenant_id = ? AND platform_aftersale_id NOT IN ?", shop.ID, shop.TenantID, ids).
			Delete(&model.ShippedRefundSuccess{}).Error
	})
}

type ServiceOrderListFilter struct {
	ShopID    uint64
	StatusTab string
	Keyword   string
	Page      int
	PageSize  int
}

func openServiceTabs() []string {
	return []string{"待处理", "处理中", "已逾期"}
}

func (r *ShopRepo) ListServiceOrders(f ServiceOrderListFilter) ([]model.ServiceOrder, int64, error) {
	q := r.db.Model(&model.ServiceOrder{}).Scopes(scopeTenant(r.tenantID))
	if f.ShopID > 0 {
		q = q.Where("shop_id = ?", f.ShopID)
	}
	if tab := strings.TrimSpace(f.StatusTab); tab != "" {
		q = q.Where("status_tab = ? OR status_tab LIKE ?", tab, "%"+tab+"%")
	} else {
		q = q.Where("status_tab IN ?", openServiceTabs())
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where(
			"platform_service_id ILIKE ? OR order_no ILIKE ? OR product_title ILIKE ? OR buyer_nick ILIKE ? OR status ILIKE ?",
			like, like, like, like, like,
		)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 200 {
		f.PageSize = 20
	}
	var list []model.ServiceOrder
	offset := (f.Page - 1) * f.PageSize
	err := q.Order("CASE WHEN deadline_at IS NULL THEN 1 ELSE 0 END, deadline_at ASC, id DESC").
		Offset(offset).Limit(f.PageSize).Find(&list).Error
	return list, total, err
}

func (r *ShopRepo) CountServiceTabs(shopID uint64) ([]struct {
	StatusTab string
	Count     int64
}, error) {
	var rows []struct {
		StatusTab string
		Count     int64
	}
	q := r.db.Model(&model.ServiceOrder{}).
		Select("status_tab as status_tab, count(*) as count").
		Scopes(scopeTenant(r.tenantID)).
		Where("status_tab IN ?", openServiceTabs())
	if shopID > 0 {
		q = q.Where("shop_id = ?", shopID)
	}
	err := q.Group("status_tab").Find(&rows).Error
	return rows, err
}

func (r *ShopRepo) UpsertServiceOrders(shop *model.MarketplaceShop, orders []model.ServiceOrder) error {
	now := time.Now()
	return r.db.Transaction(func(tx *gorm.DB) error {
		seen := make([]string, 0, len(orders))
		for i := range orders {
			o := &orders[i]
			o.TenantID = shop.TenantID
			o.ShopID = shop.ID
			o.SyncedAt = now
			sid := o.PlatformServiceID
			if sid == "" {
				continue
			}
			seen = append(seen, sid)
			var existing model.ServiceOrder
			err := tx.Where("shop_id = ? AND platform_service_id = ?", shop.ID, sid).First(&existing).Error
			if err == gorm.ErrRecordNotFound {
				if err := tx.Create(o).Error; err != nil {
					return err
				}
				continue
			}
			if err != nil {
				return err
			}
			var deadline any
			if o.DeadlineAt != nil {
				deadline = *o.DeadlineAt
			}
			if err := tx.Model(&existing).Updates(map[string]any{
				"order_no":        o.OrderNo,
				"product_title":   o.ProductTitle,
				"product_image":   o.ProductImage,
				"product_content": o.ProductContent,
				"buyer_nick":      o.BuyerNick,
				"create_source":   o.CreateSource,
				"business_type":   o.BusinessType,
				"order_type":      o.OrderType,
				"tags":            o.Tags,
				"status_tab":      o.StatusTab,
				"status":          o.Status,
				"timeout_text":    o.TimeoutText,
				"timeout_action":  o.TimeoutAction,
				"deadline_at":     deadline,
				"delay_end_time":  o.DelayEndTime,
				"detail":          o.Detail,
				"solution":        o.Solution,
				"last_log":        o.LastLog,
				"last_log_time":   o.LastLogTime,
				"create_time":     o.CreateTime,
				"raw_json":        o.RawJSON,
				"synced_at":       now,
			}).Error; err != nil {
				return err
			}
		}
		q := tx.Model(&model.ServiceOrder{}).
			Where("shop_id = ? AND status_tab IN ?", shop.ID, openServiceTabs())
		if len(seen) > 0 {
			q = q.Where("platform_service_id NOT IN ?", seen)
		}
		return q.Update("status_tab", "已离开").Error
	})
}

const defaultPluginSyncMinutes = 30

func ClampPluginSyncMinutes(n int) int {
	if n <= 0 {
		return defaultPluginSyncMinutes
	}
	if n < 5 {
		return 5
	}
	if n > 1440 {
		return 1440
	}
	return n
}

func (r *ShopRepo) PluginSyncMinutes() int {
	var item model.TenantSetting
	err := r.db.Where("tenant_id = ?", r.tenantID).First(&item).Error
	if err != nil || item.PluginSyncIntervalMin <= 0 {
		return defaultPluginSyncMinutes
	}
	return ClampPluginSyncMinutes(item.PluginSyncIntervalMin)
}

func (r *ShopRepo) SavePluginSyncMinutes(minutes int) (*model.TenantSetting, error) {
	minutes = ClampPluginSyncMinutes(minutes)
	var item model.TenantSetting
	err := r.db.Where("tenant_id = ?", r.tenantID).First(&item).Error
	if err == gorm.ErrRecordNotFound {
		item = model.TenantSetting{TenantID: r.tenantID, PluginSyncIntervalMin: minutes}
		if err := r.db.Create(&item).Error; err != nil {
			return nil, err
		}
		return &item, nil
	}
	if err != nil {
		return nil, err
	}
	item.PluginSyncIntervalMin = minutes
	if err := r.db.Save(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ShopRepo) ListTenantIDs() ([]uint64, error) {
	var ids []uint64
	err := r.db.Model(&model.MarketplaceShop{}).Distinct("tenant_id").Pluck("tenant_id", &ids).Error
	return ids, err
}
