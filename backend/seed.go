package main

import (
	"log"
	"os"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"oh-my-stock/models"
)

// ============================================================
// SeedAdmin
// 启动时如果 ADMIN_USER 不存在则创建
// 环境变量:
//   ADMIN_USER  (默认 "admin")
//   ADMIN_PASS  (默认 "admin123")
//   ADMIN_EMAIL (可选，默认 admin@local)
// ============================================================
func SeedAdmin(db *gorm.DB) {
	user := os.Getenv("ADMIN_USER")
	if user == "" {
		user = "admin"
	}
	pass := os.Getenv("ADMIN_PASS")
	if pass == "" {
		pass = "admin123"
	}
	email := os.Getenv("ADMIN_EMAIL")
	if email == "" {
		email = "admin@local"
	}

	var existing models.User
	err := db.Where("username = ?", user).First(&existing).Error
	if err == nil {
		log.Printf("👤 admin 账号已存在: %s", user)
		return
	}
	if err != gorm.ErrRecordNotFound {
		log.Printf("⚠️  seed 查询失败: %v", err)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("⚠️  seed 密码加密失败: %v", err)
		return
	}
	// 用 admin-<uuid 前 8 位> 作为 phone 占位，避免与已有空 phone 冲突
	u := models.User{
		Username:     user,
		PasswordHash: string(hash),
		Email:        email,
		Phone:        "admin-" + uuid.New().String()[:8],
		IsActive:     true,
	}
	if err := db.Create(&u).Error; err != nil {
		log.Printf("⚠️  seed 创建失败: %v", err)
		return
	}
	log.Printf("✅ 已创建 admin 账号: %s / %s", user, pass)
}
