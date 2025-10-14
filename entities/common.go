package entities

import (
	"time"
)

type DMBase struct {
	TrangThai    *int       `json:"trangThai" gorm:"column:TRANG_THAI"`
	IsDelete     *int       `json:"isDelete" gorm:"column:IS_DELETE"`
	RegDate      time.Time  `json:"regDate" gorm:"column:REG_DATE"`
	RegBy        string     `json:"regBy" gorm:"column:REG_BY"`
	LastUpdate   *time.Time `json:"lastUpdate" gorm:"column:LAST_UPDATE"`
	LastUpdateBy string     `json:"lastUpdateBy" gorm:"column:LAST_UPDATE_BY"`
}

type AddressBase struct {
	LonCenter   *float64 `json:"lonCenter" gorm:"column:LON_CENTER"`
	LatCenter   *float64 `json:"latCenter" gorm:"column:LAT_CENTER"`
	MaxLat      *float64 `json:"maxLat" gorm:"column:MAX_LAT"`
	MinLat      *float64 `json:"minLat" gorm:"column:MIN_LAT"`
	MaxLon      *float64 `json:"maxLon" gorm:"column:MAX_LON"`
	MinLon      *float64 `json:"minLon" gorm:"column:MIN_LON"`
	PolygonData *string  `json:"polygonData" gorm:"column:POLYGON_DATA;type:json"` // Polygon from CreatePolygonFromWaysAndNodes
}
type Address struct {
	ID  int64   `json:"id" gorm:"column:ID"` // OSM Node ID
	Lon float64 `json:"lon" gorm:"column:LON"`
	Lat float64 `json:"lat" gorm:"column:LAT"`
}

// WayAddress represents OSM Way data from crawled XML
type WayAddress struct {
	ID    int64    `json:"id" gorm:"column:WAY_ID"`             // OSM Way ID
	Nodes []string `json:"nodes" gorm:"column:NODES;type:json"` // Array of node references
}
