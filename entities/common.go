package entities

import (
	"time"
)

type DMBase struct {
	TrangThai    *int       `json:"trangThai" gorm:"column:TRANG_THAI"`
	IsDelete     *int       `json:"isDelete" gorm:"column:DA_XOA"`
	RegDate      time.Time  `json:"regDate" gorm:"column:NGAY_TAO"`
	RegBy        string     `json:"regBy" gorm:"column:NGUOI_TAO"`
	LastUpdate   *time.Time `json:"lastUpdate" gorm:"column:NGAY_SUA"`
	LastUpdateBy string     `json:"lastUpdateBy" gorm:"column:NGUOI_SUA"`
}

type AddressBase struct {
	Polygon   *string  `json:"polygon" gorm:"column:POLYGON_DATA"`
	LonCenter *float64 `json:"lonCenter" gorm:"column:LON_CENTER"`
	LatCenter *float64 `json:"latCenter" gorm:"column:LAT_CENTER"`

	// Max Min Lon Lat để thu hẹp xã search khi lấy polygon
	MaxLon *float64 `json:"maxLon" gorm:"column:MAX_LON"`
	MinLon *float64 `json:"minLon" gorm:"column:MIN_LON"`
	MaxLat *float64 `json:"maxLat" gorm:"column:MAX_LAT"`
	MinLat *float64 `json:"minLat" gorm:"column:MIN_LAT"`
}
type Address struct {
	ID  int64   `json:"id" gorm:"column:ID"` // OSM Node ID
	Lon float64 `json:"lon" gorm:"column:LON"`
	Lat float64 `json:"lat" gorm:"column:LAT"`
}

// WayAddress represents OSM Way data from crawled XML
type WayAddress struct {
	ID    int64    `json:"id" gorm:"column:WAY_ID"`   // OSM Way ID
	Nodes []string `json:"nodes" gorm:"column:NODES"` // Array of node references
}
