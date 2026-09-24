package repo

import "gorm.io/gorm"

type Repos struct {
	Unboxing     *UnboxingRepo
	EdgeRecord   *EdgeRecordRepo
	EdgeDevice   *EdgeDeviceRepo
	Shop         *ShopRepo
	Notification *NotificationRepo
	Issue        *IssueRepo
}

func New(db *gorm.DB) *Repos {
	return &Repos{
		Unboxing:     NewUnboxingRepo(db),
		EdgeRecord:   NewEdgeRecordRepo(db),
		EdgeDevice:   NewEdgeDeviceRepo(db),
		Shop:         NewShopRepo(db),
		Notification: NewNotificationRepo(db),
		Issue:        NewIssueRepo(db),
	}
}

func NormalizeTenantID(id uint64) uint64 {
	if id == 0 {
		return 1
	}
	return id
}

func scopeTenant(tenantID uint64) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("tenant_id = ?", NormalizeTenantID(tenantID))
	}
}
