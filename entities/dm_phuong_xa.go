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
	GhiChu         *string    `json:"-" gorm:"column:GHI_CHU"`
	TrangThai      *int       `json:"trangThai" gorm:"column:TRANG_THAI"`
	RegBy          string     `json:"regBy" gorm:"column:NGUOI_TAO"`
	RegDate        *time.Time `json:"-" gorm:"column:NGAY_TAO;<-:create"`
	LastUpdate     *time.Time `json:"-" gorm:"column:NGAY_SUA"`
	LastUpdateBy   string     `json:"lastUpdateBy" gorm:"column:NGUOI_SUA"`
	CloseDate      *time.Time `json:"-" gorm:"column:NGAY_DONG"`
	AddressBase    `gorm:"embedded" json:",inline"`
}

type DmPhuongXaTabler interface {
	TableName() string
}

func (DmPhuongXa) TableName() string {
	return "DM_PHUONG_XA"
}
