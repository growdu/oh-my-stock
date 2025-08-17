package models

import (
	"time"
)

type UserStockRule struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         string    `json:"user_id"`
	RuleName       string    `json:"rule_name"`
	RuleExpression []byte    `gorm:"type:jsonb" json:"-"` // 存 PostgreSQL JSONB
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// type UserStockRule struct {
// 	ID             int64     `gorm:"primaryKey;autoIncrement"`
// 	UserID         string    `gorm:"type:uuid;not null;index"`
// 	RuleName       string    `gorm:"type:varchar(100);not null"`
// 	RuleExpression []byte    `gorm:"type:jsonb;not null"`
// 	CreatedAt      time.Time `gorm:"autoCreateTime"`
// 	UpdatedAt      time.Time `gorm:"autoUpdateTime"`
// }

// type RuleRequest struct {
// 	UserID         string `json:"user_id" binding:"required"`
// 	RuleName       string `json:"rule_name" binding:"required"`
// 	RuleExpression []byte `json:"rule_expression" binding:"required"`
// }

// type RuleUpdateRequest struct {
// 	RuleName       string `json:"rule_name"`       // 可选，更新规则名
// 	RuleExpression []byte `json:"rule_expression"` // 可选，更新规则表达式
// }
