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

type Address struct {
	Lon float64 `json:"lon" gorm:"column:LON"`
	Lat float64 `json:"lat" gorm:"column:LAT"`
}
