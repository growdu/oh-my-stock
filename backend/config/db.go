package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Config struct {
	Database struct {
		User string `json:"user"`
		Pass string `json:"pass"`
		Host string `json:"host"`
		Port string `json:"port"`
		Name string `json:"name"`
	} `json:"database"`
}

var (
	DB     *gorm.DB
	config Config
)

func LoadConfig(path string) {
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("❌ 打开配置文件失败: %v", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		log.Fatalf("❌ 解析配置文件失败: %v", err)
	}
}

func InitDB() {

	// PostgreSQL DSN
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s port=%s dbname=%s sslmode=disable connect_timeout=5 TimeZone=Asia/Shanghai",
		config.Database.Host,
		config.Database.User,
		config.Database.Pass,
		config.Database.Port,
		config.Database.Name,
	)
	

	// 连接数据库
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ 数据库连接失败: %v", err)
	}

	DB = db
	log.Println("✅ 数据库连接成功")
}
