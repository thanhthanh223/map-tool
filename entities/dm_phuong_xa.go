package entities

import (
	"time"
)

type DmPhuongXa struct {
	MaPhuongXa     string     `json:"maPhuongXa" gorm:"column:MA_PHUONG_XA;primarykey"`
	TenPhuongXa    string     `json:"tenPhuongXa" gorm:"column:TEN_PHUONG_XA"`
	TenPhuongXaEn  string     `json:"tenPhuongXaEn" gorm:"column:TEN_PHUONG_XA_EN"`
	TrucThuocHuyen string     `json:"-" gorm:"column:TRUC_THUOC_HUYEN"`
	TrucThuocTinh  string     `json:"trucThuocTinh" gorm:"column:TRUC_THUOC_TINH"`
	ToaDoBienGioi  *string    `json:"toaDoBienGioi" gorm:"column:TOA_DO_BIEN_GIOI;type:json"` // JSON array of lon/lat coordinates
	GhiChu         *string    `json:"-" gorm:"column:GHI_CHU"`
	TrangThai      *int       `json:"trangThai" gorm:"column:TRANG_THAI"`
	RegBy          string     `json:"regBy" gorm:"column:REG_BY"`
	RegDate        *time.Time `json:"-" gorm:"column:REG_DATE;<-:create"`
	LastUpdate     *time.Time `json:"-" gorm:"column:LAST_UPDATE"`
	LastUpdateBy   string     `json:"lastUpdateBy" gorm:"column:LAST_UPDATE_BY"`
	CloseDate      *time.Time `json:"-" gorm:"column:CLOSE_DATE"`
}

type DmPhuongXaTabler interface {
	TableName() string
}

func (DmPhuongXa) TableName() string {
	return "DM_PHUONG_XA"
}
