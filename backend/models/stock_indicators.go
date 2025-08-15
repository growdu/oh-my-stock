package models

import (
	"time"
)

type StockIndicator struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	Symbol    string    `json:"symbol" gorm:"type:varchar(10);not null"` // 股票代码
	CalcDate  time.Time `json:"calc_date" gorm:"type:date;not null"`     // 计算日期
	MA5       *float64  `json:"ma5"`
	MA10      *float64  `json:"ma10"`
	MA20      *float64  `json:"ma20"`
	MA60      *float64  `json:"ma60"`
	MACD      *float64  `json:"macd"`
	DIF       *float64  `json:"dif"`
	DEA       *float64  `json:"dea"`
	K         *float64  `json:"k"`
	D         *float64  `json:"d"`
	J         *float64  `json:"j"`
	RSI6      *float64  `json:"rsi6"`
	RSI12     *float64  `json:"rsi12"`
	RSI24     *float64  `json:"rsi24"`
	BollUpper *float64  `json:"boll_upper"`
	BollMid   *float64  `json:"boll_mid"`
	BollLower *float64  `json:"boll_lower"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (StockIndicator) TableName() string {
	return "stock_indicators" // 指定表名
}
