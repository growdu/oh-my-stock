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
}

func (TargetTrendStock) TableName() string {
	return "target_trend_stock"
}
