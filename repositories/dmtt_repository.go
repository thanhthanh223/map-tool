package repositories

import (
	"fmt"
	"tool-map/entities"

	"gorm.io/gorm"
)

type DmTTRepositoryInterface interface {
	GetByName(name string) (*entities.DmTT, error)
	UpdateDataAddressByMaTT(id string, maxLat, minLat, maxLon, minLon, lonCenter, latCenter *float64) error
	UpdatePolygonDataByMaTT(id string, polygonData *string) error
	UpdatePolygonDataWithBoundsByMaTT(id string, polygonData *string, minLat, maxLat, minLon, maxLon *float64) error
	FindCommuneByCoordinate(mattChu string, lat, lon float64) (*entities.DmPhuongXa, error)
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

func (r *DmTTRepository) UpdateDataAddressByMaTT(id string, maxLat, minLat, maxLon, minLon, lonCenter, latCenter *float64) error {
	mapUpdate := map[string]interface{}{
		"MAX_LAT":    maxLat,
		"MIN_LAT":    minLat,
		"MAX_LON":    maxLon,
		"MIN_LON":    minLon,
		"LON_CENTER": lonCenter,
		"LAT_CENTER": latCenter,
	}

	if err := r.db.Model(&entities.DmTT{}).
		Where("MATT = ?", id).
		Updates(mapUpdate).Error; err != nil {
		return fmt.Errorf("failed to update boundary for DmTT %s: %w", id, err)
	}
	return nil
}

func (r *DmTTRepository) UpdatePolygonDataByMaTT(id string, polygonData *string) error {
	mapUpdate := map[string]interface{}{
		"POLYGON_DATA": polygonData,
	}
	if err := r.db.Model(&entities.DmTT{}).
		Where("MATT = ?", id).
		Updates(mapUpdate).Error; err != nil {
		return fmt.Errorf("failed to update polygon data for DmTT %s: %w", id, err)
	}
	return nil
}

func (r *DmTTRepository) UpdatePolygonDataWithBoundsByMaTT(id string, polygonData *string, minLat, maxLat, minLon, maxLon *float64) error {
	mapUpdate := map[string]interface{}{
		"POLYGON_DATA": polygonData,
		"MIN_LAT":      minLat,
		"MAX_LAT":      maxLat,
		"MIN_LON":      minLon,
		"MAX_LON":      maxLon,
	}
	if err := r.db.Model(&entities.DmTT{}).
		Where("MATT = ?", id).
		Updates(mapUpdate).Error; err != nil {
		return fmt.Errorf("failed to update polygon data with bounds for DmTT %s: %w", id, err)
	}
	return nil
}

// FindCommuneByCoordinate tìm xã/phường từ tọa độ lat/lon và mã tỉnh thành
func (r *DmTTRepository) FindCommuneByCoordinate(mattChu string, lat, lon float64) (*entities.DmPhuongXa, error) {
	var commune entities.DmPhuongXa

	// Bước 1: Filter bằng bounding box (nhanh)
	// Bước 2: Kiểm tra point-in-polygon (chính xác)
	query := `
		SELECT * FROM DMPHUONGXA 
		WHERE TRUCTHUOCTINH = ? 
		AND MIN_LAT <= ? AND MAX_LAT >= ?
		AND MIN_LON <= ? AND MAX_LON >= ?
		AND POLYGON_DATA IS NOT NULL
	`

	if err := r.db.Raw(query, mattChu, lat, lat, lon, lon).Scan(&commune).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find commune by coordinate: %w", err)
	}

	// TODO: Implement point-in-polygon check using POLYGON_DATA
	// For now, return the first match (can be improved with proper spatial query)

	return &commune, nil
}
