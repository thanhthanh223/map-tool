package repositories

import (
	"fmt"
	"log"
	"strings"
	"tool-map/entities"
	"tool-map/util"

	"gorm.io/gorm"
)

type DmPhuongXaRepositoryInterface interface {
	GetByName(name string, maTT string) (*entities.DmPhuongXa, error)
	GetWhenHavePolygonAndCenterNull() ([]entities.DmPhuongXa, error)

	UpdateDataAddressByMaPhuongXa(id string, maxLat, minLat, maxLon, minLon, lonCenter, latCenter *float64) error
	UpdatePolygonDataByMaPhuongXa(id string, polygonData *string) error
	UpdateLatLonCenterByMaPhuongXa(id string, latCenter, lonCenter *float64) error
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

func (r *DmPhuongXaRepository) GetByName(name string, maTT string) (*entities.DmPhuongXa, error) {
	var dmPhuongXa entities.DmPhuongXa
	// Thử tìm kiếm chính xác trước
	if err := r.db.Where("TEN_PHUONG_XA = ? AND TRUC_THUOC_TINH = ? AND POLYGON_DATA IS NULL", name, maTT).First(&dmPhuongXa).Error; err != nil {
		// Lấy ra toàn bộ phường xã thuộc tỉnh theo mã tỉnh, chỉ lấy name và mã phường xã
		var phuongs []struct {
			MaPhuongXa  string
			TenPhuongXa string
		}

		if err := r.db.
			Table("DM_PHUONG_XA").
			Select("MA_PHUONG_XA, TEN_PHUONG_XA").
			Where("TRUC_THUOC_TINH = ?", maTT).
			Where("POLYGON_DATA IS NULL").
			Find(&phuongs).Error; err != nil {
			log.Printf("Lỗi khi lấy ra toàn bộ phường xã thuộc tỉnh theo mã tỉnh: %v", err)
			return nil, err
		}

		for _, phuong := range phuongs {
			if util.RemoveVietnameseAccent(strings.ToLower(phuong.TenPhuongXa)) == util.RemoveVietnameseAccent(strings.ToLower(name)) {
				dmPhuongXa.MaPhuongXa = phuong.MaPhuongXa
				dmPhuongXa.TenPhuongXa = phuong.TenPhuongXa

				err := r.db.Where("MA_PHUONG_XA = ? AND POLYGON_DATA IS NULL", dmPhuongXa.MaPhuongXa).First(&dmPhuongXa).Error
				if err != nil {
					log.Printf("Lỗi khi lấy ra phường xã từ database: %v", err)
					return nil, err
				}
				return &dmPhuongXa, nil
			}
		}
	}
	return &dmPhuongXa, nil
}

func (r *DmPhuongXaRepository) GetWhenHavePolygonAndCenterNull() ([]entities.DmPhuongXa, error) {
	var dmPhuongXas []entities.DmPhuongXa
	// In SQL, equality should be a single '='. ORA-00936: missing expression likely due to '==' instead of '='.
	if err := r.db.Where("POLYGON_DATA IS NOT NULL AND LAT_CENTER = 0 AND LON_CENTER = 0").Find(&dmPhuongXas).Error; err != nil {
		return nil, err
	}
	return dmPhuongXas, nil
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

func (r *DmPhuongXaRepository) UpdateLatLonCenterByMaPhuongXa(id string, latCenter, lonCenter *float64) error {
	mapUpdate := map[string]interface{}{
		"LAT_CENTER": latCenter,
		"LON_CENTER": lonCenter,
	}
	if err := r.db.Model(&entities.DmPhuongXa{}).
		Where("MA_PHUONG_XA = ?", id).
		Updates(mapUpdate).Error; err != nil {
		return fmt.Errorf("failed to update lat lon center for DmPhuongXa %s: %w", id, err)
	}
	return nil
}
