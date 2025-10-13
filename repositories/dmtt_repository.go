package repositories

import (
	"fmt"
	"tool-map/entities"

	"gorm.io/gorm"
)

type DmTTRepositoryInterface interface {
	GetByName(name string) (*entities.DmTT, error)
	UpdateDataAddressByMaTT(id string, bienGioi, wayAddress *string, lonCenter, latCenter *float64) error
}
type DmTTRepository struct {
	*BaseRepository
}

// NewDmTTRepository creates a new DmTT repository
func NewDmTTRepository(db *gorm.DB) *DmTTRepository {
	return &DmTTRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// Create creates a new DmTT record
func (r *DmTTRepository) Create(dmTT *entities.DmTT) error {
	if err := r.db.Create(dmTT).Error; err != nil {
		return fmt.Errorf("failed to create DmTT: %w", err)
	}
	return nil
}

func (r *DmTTRepository) GetByName(name string) (*entities.DmTT, error) {
	var dmTT entities.DmTT
	if err := r.db.Where("TENTT LIKE ?", "%"+name+"%").First(&dmTT).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &dmTT, nil
}

func (r *DmTTRepository) UpdateDataAddressByMaTT(id string, bienGioi, wayAddress *string, lonCenter, latCenter *float64) error {
	mapUpdate := map[string]interface{}{
		"TOA_DO_BIEN_GIOI": bienGioi,
		"WAY_ADDRESS":      wayAddress,
		"LON_CENTER":       lonCenter,
		"LAT_CENTER":       latCenter,
	}
	if err := r.db.Model(&entities.DmTT{}).
		Where("MATT = ?", id).
		Updates(mapUpdate).Error; err != nil {
		return fmt.Errorf("failed to update boundary for DmTT %s: %w", id, err)
	}
	return nil
}
