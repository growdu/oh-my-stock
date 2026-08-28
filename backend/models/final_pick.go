package models

import "time"

// FinalPick 进阶精选 Top N 落库（每日精选结果留存，便于回看）
type FinalPick struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TradeDate time.Time `gorm:"type:date;not null;index" json:"trade_date"`
	Rank      int       `gorm:"not null"                  json:"rank"`
	Symbol    string    `gorm:"type:varchar(10);not null" json:"symbol"`
	Name      string    `gorm:"type:varchar(50);not null" json:"name"`
	Industry  string    `gorm:"type:varchar(50)"          json:"industry"`
	Market    string    `gorm:"type:varchar(20)"          json:"market"`
	Score     int       `gorm:"not null"                  json:"score"`
	Breakdown string    `gorm:"type:jsonb;not null"       json:"breakdown"`     // 6 维明细
	Matched   string    `gorm:"type:text;not null;default '[]'" json:"matched"` // 命中的预设 id（JSON 数组）
	CreatedAt time.Time `gorm:"autoCreateTime"            json:"created_at"`
}

// TableName overrides the table name used by FinalPick to `final_picks`
func (FinalPick) TableName() string { return "final_picks" }
