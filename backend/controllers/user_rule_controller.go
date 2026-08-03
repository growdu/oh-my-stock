package controllers

import (
	"encoding/json"
	"net/http"
	"oh-my-stock/config"
	"oh-my-stock/middleware"
	"oh-my-stock/models"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type RuleData struct {
	ID             uint                   `json:"id"`
	UserID         string                 `json:"user_id"`
	RuleName       string                 `json:"rule_name"`
	RuleExpression map[string]interface{} `json:"rule_expression"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

type GetRulesResponse struct {
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
	Total    int64      `json:"total"`
	Data     []RuleData `json:"data"`
}

// AddRule 新增规则
func AddRule(c *gin.Context) {
	uid := middleware.GetUserID(c)
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	var req struct {
		RuleName       string                 `json:"rule_name" binding:"required"`
		RuleExpression map[string]interface{} `json:"rule_expression" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	exprJSON, _ := json.Marshal(req.RuleExpression)
	rule := models.UserStockRule{
		UserID:         uid,
		RuleName:       req.RuleName,
		RuleExpression: exprJSON,
	}
	if err := config.DB.Create(&rule).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "创建规则失败，可能是同名规则已存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "规则创建成功", "rule_id": rule.ID})
}

// GetRules 分页获取规则
func GetRules(c *gin.Context) {
	uid := middleware.GetUserID(c)
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	var total int64
	var rules []models.UserStockRule
	q := config.DB.Model(&models.UserStockRule{}).Where("user_id = ?", uid)
	q.Count(&total)
	q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rules)

	resp := make([]RuleData, 0, len(rules))
	for _, r := range rules {
		var expr map[string]interface{}
		_ = json.Unmarshal(r.RuleExpression, &expr)
		resp = append(resp, RuleData{
			ID:             r.ID,
			UserID:         r.UserID,
			RuleName:       r.RuleName,
			RuleExpression: expr,
			CreatedAt:      r.CreatedAt,
			UpdatedAt:      r.UpdatedAt,
		})
	}
	c.JSON(http.StatusOK, GetRulesResponse{
		Page: page, PageSize: pageSize, Total: total, Data: resp,
	})
}

// UpdateRule
func UpdateRule(c *gin.Context) {
	uid := middleware.GetUserID(c)
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID 无效"})
		return
	}
	var req struct {
		RuleName       string                 `json:"rule_name"`
		RuleExpression map[string]interface{} `json:"rule_expression"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var rule models.UserStockRule
	if err := config.DB.Where("id = ? AND user_id = ?", id, uid).First(&rule).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "规则不存在"})
		return
	}
	if req.RuleName != "" {
		rule.RuleName = req.RuleName
	}
	if req.RuleExpression != nil {
		b, _ := json.Marshal(req.RuleExpression)
		rule.RuleExpression = b
	}
	rule.UpdatedAt = time.Now()
	if err := config.DB.Save(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}
	var expr map[string]interface{}
	_ = json.Unmarshal(rule.RuleExpression, &expr)
	c.JSON(http.StatusOK, gin.H{"message": "更新成功", "rule": RuleData{
		ID: rule.ID, UserID: rule.UserID, RuleName: rule.RuleName,
		RuleExpression: expr, CreatedAt: rule.CreatedAt, UpdatedAt: rule.UpdatedAt,
	}})
}

// DeleteRule
func DeleteRule(c *gin.Context) {
	uid := middleware.GetUserID(c)
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID 无效"})
		return
	}
	res := config.DB.Where("id = ? AND user_id = ?", id, uid).Delete(&models.UserStockRule{})
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "规则不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "规则删除成功"})
}
