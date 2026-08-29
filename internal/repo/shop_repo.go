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
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where(
			"platform_aftersale_id ILIKE ? OR order_no ILIKE ? OR product_title ILIKE ? OR status ILIKE ? OR return_logistics_no ILIKE ?",
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
	var list []model.AftersaleTicket
	offset := (f.Page - 1) * f.PageSize
	err := q.Preload("CardKeys").Order("id DESC").Offset(offset).Limit(f.PageSize).Find(&list).Error
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
				if err := tx.Model(&existing).Updates(map[string]any{
					"order_no":            t.OrderNo,
					"product_title":       t.ProductTitle,
					"product_image":       t.ProductImage,
					"sku":                 t.SKU,
					"product_tags":        t.ProductTags,
					"tags":                t.Tags,
					"qty":                 t.Qty,
					"buy_qty":             t.BuyQty,
					"pay_amount":          t.PayAmount,
					"refund_amount":       t.RefundAmount,
					"aftersale_type":      t.AftersaleType,
					"reason":              t.Reason,
					"status":              t.Status,
					"timeout_text":        t.TimeoutText,
					"timeout_action":      t.TimeoutAction,
					"deadline_at":         deadline,
					"dispute":             t.Dispute,
					"logistics":           t.Logistics,
					"return_logistics_no": t.ReturnLogisticsNo,
					"apply_time":          t.ApplyTime,
					"raw_json":            t.RawJSON,
					"synced_at":           now,
				}).Error; err != nil {
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
		return q.Delete(&model.AftersaleTicketCard{}).Error
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

func (r *ShopRepo) ListTenantIDs() ([]uint64, error) {
	var ids []uint64
	err := r.db.Model(&model.MarketplaceShop{}).Distinct("tenant_id").Pluck("tenant_id", &ids).Error
	return ids, err
}
