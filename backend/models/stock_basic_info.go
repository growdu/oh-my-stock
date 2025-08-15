package models

import "time"

// StockBasicInfo 股票基础信息
type StockBasicInfo struct {
	ID                uint       `gorm:"primaryKey;autoIncrement" json:"id" example:"1"`
	Symbol            string     `gorm:"type:varchar(10);unique;not null" json:"symbol" example:"600000"`
	Name              string     `gorm:"type:varchar(50);not null" json:"name" example:"浦发银行"`
	FullName          string     `gorm:"type:varchar(100)" json:"full_name" example:"上海浦东发展银行股份有限公司"`
	Industry          string     `gorm:"type:varchar(50)" json:"industry" example:"银行"`
	Area              string     `gorm:"type:varchar(50)" json:"area" example:"上海"`
	Market            string     `gorm:"type:varchar(20)" json:"market" example:"主板"`
	ListingDate       *time.Time `json:"listing_date" example:"1999-11-10"`
	OutstandingShares float64    `gorm:"type:decimal(20,4)" json:"outstanding_shares" example:"2930026.0000"`
	TotalShares       float64    `gorm:"type:decimal(20,4)" json:"total_shares" example:"2930026.0000"`
	IsHs              bool       `json:"is_hs" example:"true"`
	Status            string     `gorm:"type:varchar(20)" json:"status" example:"上市"`
	CreatedAt         time.Time  `gorm:"autoCreateTime" json:"created_at" example:"2025-08-13T10:00:00+08:00"`
	UpdatedAt         time.Time  `gorm:"autoUpdateTime" json:"updated_at" example:"2025-08-13T10:00:00+08:00"`
}

func (StockBasicInfo) TableName() string {
	return "stock_basic_info"
}
