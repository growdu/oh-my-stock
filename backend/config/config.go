package config

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ============================================================
// 配置结构
// ============================================================
type DBConfig struct {
	User string `json:"user"`
	Pass string `json:"pass"`
	Host string `json:"host"`
	Port string `json:"port"`
	Name string `json:"name"`
}

type FrontendConfig struct {
	Origin string `json:"origin"`
}

type JWTConfig struct {
	Secret  string `json:"secret"`
	TTLHours int    `json:"ttl_hours"`
}

type ServerConfig struct {
	Host string `json:"host"`
	Port string `json:"port"`
}

type Config struct {
	Database DBConfig      `json:"database"`
	Frontend FrontendConfig `json:"frontend"`
	JWT      JWTConfig     `json:"jwt"`
	Server   ServerConfig  `json:"server"`
}

var (
	Cfg  Config
	DB   *gorm.DB
)

// ============================================================
// 加载配置（支持 ${ENV_VAR} 占位符，从环境变量读，缺失则保留原值）
// ============================================================
func LoadConfig(path string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("❌ 打开配置文件失败: %v", err)
	}
	expanded := expandEnv(string(raw))

	if err := json.Unmarshal([]byte(expanded), &Cfg); err != nil {
		log.Fatalf("❌ 解析配置文件失败: %v", err)
	}

	// 必要的默认值
	if Cfg.Server.Host == "" {
		Cfg.Server.Host = "0.0.0.0"
	}
	if Cfg.Server.Port == "" {
		Cfg.Server.Port = "3003"
	}
	if Cfg.JWT.TTLHours <= 0 {
		Cfg.JWT.TTLHours = 168 // 默认 7 天
	}
	log.Printf("✅ 配置加载完成: db=%s/%s frontend=%s jwt_ttl=%dh",
		Cfg.Database.Host, Cfg.Database.Name, Cfg.Frontend.Origin, Cfg.JWT.TTLHours)
}

func expandEnv(s string) string {
	return os.Expand(s, func(k string) string { return os.Getenv(k) })
}

func GetFrontOrigin() string { return Cfg.Frontend.Origin }
func GetServerPort() string { return Cfg.Server.Port }
func GetServerHost() string { return Cfg.Server.Host }

// ============================================================
// 初始化数据库
// ============================================================
func InitDB() {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s port=%s dbname=%s sslmode=disable connect_timeout=5 TimeZone=Asia/Shanghai",
		Cfg.Database.Host,
		Cfg.Database.User,
		Cfg.Database.Pass,
		Cfg.Database.Port,
		Cfg.Database.Name,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ 数据库连接失败: %v", err)
	}
	DB = db
	log.Println("✅ 数据库连接成功")
}

// ============================================================
// JWT (HMAC-SHA256，自实现，不引第三方库)
//
// token 格式: base64url(payload).hex(hmac_sha256(secret, base64url(payload)))
// payload    : { "uid": <user_uuid>, "exp": <unix_ts>, "iat": <unix_ts> }
// ============================================================
type jwtPayload struct {
	UID string `json:"uid"`
	IAT int64  `json:"iat"`
	EXP int64  `json:"exp"`
}

func IssueToken(userID string) (string, error) {
	now := time.Now().Unix()
	p := jwtPayload{
		UID: userID,
		IAT: now,
		EXP: now + int64(Cfg.JWT.TTLHours)*3600,
	}
	payloadJSON, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	mac := hmac.New(sha256.New, []byte(Cfg.JWT.Secret))
	mac.Write([]byte(payloadB64))
	sig := hex.EncodeToString(mac.Sum(nil))
	return payloadB64 + "." + sig, nil
}

func VerifyToken(token string) (string, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("malformed token")
	}
	payloadB64, sig := parts[0], parts[1]
	mac := hmac.New(sha256.New, []byte(Cfg.JWT.Secret))
	mac.Write([]byte(payloadB64))
	want := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sig), []byte(want)) {
		return "", fmt.Errorf("bad signature")
	}
	raw, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return "", err
	}
	var p jwtPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return "", err
	}
	if p.EXP > 0 && time.Now().Unix() > p.EXP {
		return "", fmt.Errorf("token expired")
	}
	return p.UID, nil
}

// 辅助：从环境变量读整型（供 main 调用端口 log 用）
func EnvInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
