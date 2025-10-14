package repositories

import (
	"fmt"
	"tool-map/entities"

	"gorm.io/gorm"
)

type DmPhuongXaRepositoryInterface interface {
	GetByName(name string) (*entities.DmPhuongXa, error)
	UpdateDataAddressByMaPhuongXa(id string, maxLat, minLat, maxLon, minLon, lonCenter, latCenter *float64) error
	UpdatePolygonDataByMaPhuongXa(id string, polygonData *string) error
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

func (r *DmPhuongXaRepository) UpdateDataAddressByMaPhuongXa(id string, maxLat, minLat, maxLon, minLon, lonCenter, latCenter *float64) error {
	mapUpdate := map[string]interface{}{
		"MAX_LAT":    maxLat,
		"MIN_LAT":    minLat,
		"MAX_LON":    maxLon,
		"MIN_LON":    minLon,
		"LON_CENTER": lonCenter,
		"LAT_CENTER": latCenter,
	}
	if err := r.db.Model(&entities.DmPhuongXa{}).
		Where("MA_PHUONG_XA = ?", id).
		Updates(mapUpdate).Error; err != nil {
		return fmt.Errorf("failed to update boundary for DmPhuongXa %s: %w", id, err)
	}
	return nil
}

func (r *DmPhuongXaRepository) UpdatePolygonDataByMaPhuongXa(id string, polygonData *string) error {
	mapUpdate := map[string]interface{}{
		"POLYGON_DATA": polygonData,
	}
	if err := r.db.Model(&entities.DmPhuongXa{}).
		Where("MA_PHUONG_XA = ?", id).
		Updates(mapUpdate).Error; err != nil {
		return fmt.Errorf("failed to update polygon data for DmPhuongXa %s: %w", id, err)
	}
	return nil
}
