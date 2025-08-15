package models

import (
	"time"
)

// StockDailyData 股票日线数据
type StockDailyData struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Symbol        string    `gorm:"type:varchar(10);not null" json:"symbol"`  // 股票代码
	TradeDate     time.Time `gorm:"type:date;not null" json:"trade_date"`     // 交易日期
	Open          float64   `gorm:"type:decimal(12,4)" json:"open"`           // 开盘价
	High          float64   `gorm:"type:decimal(12,4)" json:"high"`           // 最高价
	Low           float64   `gorm:"type:decimal(12,4)" json:"low"`            // 最低价
	Close         float64   `gorm:"type:decimal(12,4)" json:"close"`          // 收盘价
	AdjClose      float64   `gorm:"type:decimal(12,4)" json:"adj_close"`      // 后复权收盘价
	Volume        int64     `gorm:"type:bigint" json:"volume"`                // 成交量(股)
	Turnover      float64   `gorm:"type:decimal(20,4)" json:"turnover"`       // 成交额(元)
	ChangePercent float64   `gorm:"type:decimal(10,4)" json:"change_percent"` // 涨跌幅(%)
	ChangeAmount  float64   `gorm:"type:decimal(10,4)" json:"change_amount"`  // 涨跌额
	TurnoverRate  float64   `gorm:"type:decimal(10,4)" json:"turnover_rate"`  // 换手率(%)
	PETTM         float64   `gorm:"type:decimal(10,4)" json:"pe_ttm"`         // 市盈率(TTM)
	PB            float64   `gorm:"type:decimal(10,4)" json:"pb"`             // 市净率
	Amplitude     float64   `gorm:"type:decimal(10,4)" json:"amplitude"`      // 振幅(%)
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`         // 创建时间
}

func (StockDailyData) TableName() string {
	return "stock_daily_data" // 指定表名
}
