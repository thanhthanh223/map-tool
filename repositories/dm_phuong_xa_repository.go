package repositories

import (
	"fmt"
	"tool-map/entities"

	"gorm.io/gorm"
)

type DmPhuongXaRepositoryInterface interface {
	GetByName(name string) (*entities.DmPhuongXa, error)
	UpdateBienGioiByMaPhuongXa(id string, bienGioi *string) error
}

// DmPhuongXaRepository handles database operations for DmPhuongXa entities
type DmPhuongXaRepository struct {
	*BaseRepository
}

// NewDmPhuongXaRepository creates a new DmPhuongXa repository
func NewDmPhuongXaRepository(db *gorm.DB) *DmPhuongXaRepository {
	return &DmPhuongXaRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

func (r *DmPhuongXaRepository) GetByName(name string) (*entities.DmPhuongXa, error) {
	var dmPhuongXa entities.DmPhuongXa
	if err := r.db.Where("TEN_PHUONG_XA = ?", name).First(&dmPhuongXa).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &dmPhuongXa, nil
}

func (r *DmPhuongXaRepository) UpdateBienGioiByMaPhuongXa(id string, bienGioi *string) error {
	if err := r.db.Model(&entities.DmPhuongXa{}).
		Where("MA_PHUONG_XA = ?", id).
		Update("TOA_DO_BIEN_GIOI", bienGioi).Error; err != nil {
		return fmt.Errorf("failed to update boundary for DmPhuongXa %s: %w", id, err)
	}
	return nil
}
