package entities

type DmTT struct {
	MaTT        string `json:"maTT" gorm:"column:MATT;primarykey"`
	TenTT       string `json:"tenTT" gorm:"column:TENTT"`
	TenTTEn     string `json:"tenTTEn" gorm:"column:TENTT_EN"`
	MaTTChu     string `json:"maTTChu" gorm:"column:MATT_CHU"`
	MoTa        string `json:"moTa" gorm:"column:MO_TA"`
	AddressBase `gorm:"embedded" json:",inline"`
	DMBase      `gorm:"embedded" json:",inline"`
	DmPhuongXa  []DmPhuongXa `json:"-" gorm:"foreignKey:TrucThuocTinh;->"`
}

type DmTTTabler interface {
	TableName() string
}

func (DmTT) TableName() string {
	return "DMTT"
}
