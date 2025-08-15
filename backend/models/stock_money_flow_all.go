package models

import (
	"time"
)

type StockMoneyFlowAll struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	TimeSpan      int       `gorm:"column:time_span;not null" json:"time_span"`
	SerialNumber  *int      `gorm:"column:serial_number" json:"serial_number"`
	Symbol        string    `gorm:"column:symbol;size:10;not null" json:"symbol"`
	Name          string    `gorm:"column:name;size:50" json:"name"`
	LatestPrice   *float64  `gorm:"column:latest_price" json:"latest_price"`
	ChangePercent *float64  `gorm:"column:change_percent" json:"change_percent"`
	TurnoverRate  *float64  `gorm:"column:turnover_rate" json:"turnover_rate"`
	InflowAmount  *float64  `gorm:"column:inflow_amount" json:"inflow_amount"`
	OutflowAmount *float64  `gorm:"column:outflow_amount" json:"outflow_amount"`
	NetAmount     *float64  `gorm:"column:net_amount" json:"net_amount"`
	Turnover      *float64  `gorm:"column:turnover" json:"turnover"`
	TradeDate     time.Time `gorm:"column:trade_date;not null" json:"trade_date"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (StockMoneyFlowAll) TableName() string {
	return "stock_money_flow_all"
}
