package models

import "time"

type StockHistory struct {
	Symbol        string    `json:"symbol"`
	Name          string    `json:"name"`
	TradeDate     time.Time `json:"trade_date"`
	Open          float64   `json:"open"`
	Close         float64   `json:"close"`
	High          float64   `json:"high"`
	Low           float64   `json:"low"`
	Volume        float64   `json:"volume"`
	TurnoverRate  float64   `json:"turnover_rate"`
	ChangePercent float64   `json:"change_percent"`

	InflowAmount  float64 `json:"in_amount"`
	OutflowAmount float64 `json:"out_amount"`
	NetAmount     float64 `json:"net_amount"`
	Turnover      float64 `json:"turnover"`
}

func (StockHistory) TableName() string {
	return "stock_history_mv" // 指定表名
}
