package models

import (
	"time"
)

type UserStockRule struct {
	ID             int64                  `gorm:"primaryKey;autoIncrement"`
	UserID         string                 `gorm:"type:uuid;not null;index"`
	RuleName       string                 `gorm:"type:varchar(100);not null"`
	RuleExpression map[string]interface{} `gorm:"type:jsonb;not null"`
	CreatedAt      time.Time              `gorm:"autoCreateTime"`
	UpdatedAt      time.Time              `gorm:"autoUpdateTime"`
}

type RuleRequest struct {
	UserID         string                 `json:"user_id" binding:"required"`
	RuleName       string                 `json:"rule_name" binding:"required"`
	RuleExpression map[string]interface{} `json:"rule_expression" binding:"required"`
}

type RuleUpdateRequest struct {
	RuleName       string                 `json:"rule_name"`       // 可选，更新规则名
	RuleExpression map[string]interface{} `json:"rule_expression"` // 可选，更新规则表达式
}
