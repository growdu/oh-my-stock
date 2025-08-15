package models

import (
	"time"
)

type StockMoneyFlow struct {
	ID               uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	Symbol           string    `json:"symbol" gorm:"type:varchar(10);not null"` // 股票代码
	TradeDate        time.Time `json:"trade_date" gorm:"type:date;not null"`    // 交易日期
	MainNet          *float64  `json:"main_net"`                                // 主力净流入
	RetailNet        *float64  `json:"retail_net"`                              // 散户净流入
	LargeOrderRatio  *float64  `json:"large_order_ratio"`                       // 大单成交占比(%)
	MediumOrderRatio *float64  `json:"medium_order_ratio"`                      // 中单成交占比(%)
	SmallOrderRatio  *float64  `json:"small_order_ratio"`                       // 小单成交占比(%)
	CreatedAt        time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (StockMoneyFlow) TableName() string {
	return "stock_money_flow" // 指定表名
}
