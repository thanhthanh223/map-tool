package repositories

import (
	"fmt"
	"tool-map/entities"

	"gorm.io/gorm"
)

type DmTTRepositoryInterface interface {
	GetByName(name string) (*entities.DmTT, error)
	UpdateBienGioiByMaTT(id string, bienGioi *string) error
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
	if err := r.db.Where("TEN_TT = ?", name).First(&dmTT).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &dmTT, nil
}

func (r *DmTTRepository) UpdateBienGioiByMaTT(id string, bienGioi *string) error {
	if err := r.db.Model(&entities.DmTT{}).
		Where("MA_TT = ?", id).
		Update("TOA_DO_BIEN_GIOI", bienGioi).Error; err != nil {
		return fmt.Errorf("failed to update boundary for DmTT %s: %w", id, err)
	}
	return nil
}
