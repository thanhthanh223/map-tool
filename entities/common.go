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
	ToaDoBienGioi *string  `json:"toaDoBienGioi" gorm:"column:TOA_DO_BIEN_GIOI;type:json"` // JSON array of lon/lat coordinates
	LonCenter     *float64 `json:"lonCenter" gorm:"column:LON_CENTER"`
	LatCenter     *float64 `json:"latCenter" gorm:"column:LAT_CENTER"`
}
type Address struct {
	Lon float64 `json:"lon" gorm:"column:LON"`
	Lat float64 `json:"lat" gorm:"column:LAT"`
}
