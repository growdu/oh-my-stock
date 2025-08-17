package models

import (
	"time"
)

type UserFavoriteStock struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	UserID    string    `gorm:"type:uuid;not null;index"` // 关联 users.id
	Symbol    string    `gorm:"type:varchar(20);not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

type FavoriteRequest struct {
    UserID string `json:"user_id" binding:"required"`
    Symbol string `json:"symbol" binding:"required"`
}