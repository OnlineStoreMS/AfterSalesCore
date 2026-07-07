package repo

import (
	"aftersalescore/internal/model"

	"gorm.io/gorm"
)

type EdgeDeviceRepo struct {
	db *gorm.DB
}

func NewEdgeDeviceRepo(db *gorm.DB) *EdgeDeviceRepo {
	return &EdgeDeviceRepo{db: db}
}

func (r *EdgeDeviceRepo) List() ([]model.EdgeDevice, error) {
	var list []model.EdgeDevice
	err := r.db.Order("edge_id ASC").Find(&list).Error
	return list, err
}

func (r *EdgeDeviceRepo) Get(id uint64) (*model.EdgeDevice, error) {
	var dev model.EdgeDevice
	err := r.db.First(&dev, id).Error
	if err != nil {
		return nil, err
	}
	return &dev, nil
}

func (r *EdgeDeviceRepo) GetByEdgeID(edgeID string) (*model.EdgeDevice, error) {
	var dev model.EdgeDevice
	err := r.db.Where("edge_id = ?", edgeID).First(&dev).Error
	if err != nil {
		return nil, err
	}
	return &dev, nil
}

func (r *EdgeDeviceRepo) Create(dev *model.EdgeDevice) error {
	return r.db.Create(dev).Error
}

func (r *EdgeDeviceRepo) Save(dev *model.EdgeDevice) error {
	return r.db.Save(dev).Error
}

func (r *EdgeDeviceRepo) Delete(id uint64) error {
	return r.db.Delete(&model.EdgeDevice{}, id).Error
}

func (r *EdgeDeviceRepo) ListProbeable() ([]model.EdgeDevice, error) {
	var list []model.EdgeDevice
	err := r.db.Where("base_url <> ''").Find(&list).Error
	return list, err
}

func (r *EdgeDeviceRepo) UpsertByEdgeID(edgeID, name string) error {
	var dev model.EdgeDevice
	err := r.db.Where("edge_id = ?", edgeID).First(&dev).Error
	if err == gorm.ErrRecordNotFound {
		return r.db.Create(&model.EdgeDevice{
			EdgeID: edgeID,
			Name:   name,
			Status: model.EdgeDeviceStatusUnknown,
		}).Error
	}
	if err != nil {
		return err
	}
	if name != "" && dev.Name != name {
		dev.Name = name
		return r.db.Save(&dev).Error
	}
	return nil
}
