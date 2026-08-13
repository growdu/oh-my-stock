package models

import (
	"time"
)

// TargetTrendStock 候选股（规则执行结果）
type TargetTrendStock struct {
	ID            uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Symbol        string    `gorm:"type:varchar(10);not null;uniqueIndex:idx_target_uniq" json:"symbol"`
	Name          string    `gorm:"type:varchar(50)" json:"name"`
	RuleName      string    `gorm:"type:varchar(100);uniqueIndex:idx_target_uniq" json:"rule_name"`
	RuleID        *uint     `gorm:"uniqueIndex:idx_target_uniq" json:"rule_id"`
	UserID        string    `gorm:"type:uuid" json:"user_id"`
	CurrentPrice  float64   `gorm:"type:decimal(12,4)" json:"current_price"`
	Change3D      float64   `gorm:"type:decimal(10,4);column:change_3d" json:"change_3d"`
	ChangePercent float64   `gorm:"type:decimal(10,4)" json:"change_percent"`
	TurnoverRate  float64   `gorm:"type:decimal(10,4)" json:"turnover_rate"`
	NetInflow     float64   `gorm:"type:decimal(20,4)" json:"net_inflow"`
	Industry      string    `gorm:"type:varchar(50)" json:"industry"`
	Market        string    `gorm:"type:varchar(20)" json:"market"`
	MatchedAt     time.Time `gorm:"type:date;not null" json:"matched_at"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`

	// 技术指标快照：规则命中时点的 MA / MACD / KDJ / RSI / BOLL 当前值 + 昨日 lag
	// 这些字段对前端展示有意义（命中后回看历史快照用）
	MA5       float64 `gorm:"type:decimal(12,4)" json:"ma5"`
	MA10      float64 `gorm:"type:decimal(12,4)" json:"ma10"`
	MA20      float64 `gorm:"type:decimal(12,4)" json:"ma20"`
	MA60      float64 `gorm:"type:decimal(12,4)" json:"ma60"`
	MA5Prev   float64 `gorm:"type:decimal(12,4);column:ma5_prev" json:"ma5_prev"`
	MA10Prev  float64 `gorm:"type:decimal(12,4);column:ma10_prev" json:"ma10_prev"`
	MA20Prev  float64 `gorm:"type:decimal(12,4);column:ma20_prev" json:"ma20_prev"`
	MACD      float64 `gorm:"type:decimal(12,4)" json:"macd"`
	DIF       float64 `gorm:"type:decimal(12,4)" json:"dif"`
	DEA       float64 `gorm:"type:decimal(12,4)" json:"dea"`
	K         float64 `gorm:"type:decimal(12,4)" json:"k"`
	D         float64 `gorm:"type:decimal(12,4)" json:"d"`
	J         float64 `gorm:"type:decimal(12,4)" json:"j"`
	RSI6      float64 `gorm:"type:decimal(12,4);column:rsi6" json:"rsi6"`
	RSI12     float64 `gorm:"type:decimal(12,4);column:rsi12" json:"rsi12"`
	RSI24     float64 `gorm:"type:decimal(12,4);column:rsi24" json:"rsi24"`
	BollUpper float64 `gorm:"type:decimal(12,4);column:boll_upper" json:"boll_upper"`
	BollMid   float64 `gorm:"type:decimal(12,4);column:boll_mid" json:"boll_mid"`
	BollLower float64 `gorm:"type:decimal(12,4);column:boll_lower" json:"boll_lower"`

	// 财报快照（最新一期）：净利润 / 净利同比 / 营收同比
	// 用 *float64 保留 "无数据" vs "0" 的区分（COALESCE 会把 NULL 吞掉）
	NetProfit     *float64 `gorm:"type:decimal(20,4);column:net_profit"      json:"net_profit,omitempty"`
	NetProfitYoy  *float64 `gorm:"type:decimal(10,4);column:net_profit_yoy"  json:"net_profit_yoy,omitempty"`
	RevenueYoy    *float64 `gorm:"type:decimal(10,4);column:revenue_yoy"     json:"revenue_yoy,omitempty"`
}

func (TargetTrendStock) TableName() string {
	return "target_trend_stock"
}
