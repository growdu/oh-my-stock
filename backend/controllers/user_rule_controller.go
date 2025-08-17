package controllers

import (
	"encoding/json"
	"net/http"
	"oh-my-stock/config"
	"oh-my-stock/models"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// ======== 添加选股规则 ========
type AddRuleRequest struct {
	UserID         string                 `json:"user_id" binding:"required" example:"1001"`
	RuleName       string                 `json:"rule_name" binding:"required" example:"突破均线"`
	RuleExpression map[string]interface{} `json:"rule_expression" binding:"required" example:"{\"ma5_gt_ma10\": true}"`
}

// AddRule 添加规则
// @Summary 添加选股规则
// @Description 用户添加选股规则
// @Tags 用户选股规则
// @Accept json
// @Produce json
// @Param rule body AddRuleRequest true "规则信息"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /user/rules [post]
func AddRule(c *gin.Context) {
	var req AddRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	exprJSON, err := json.Marshal(req.RuleExpression)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "规则表达式无效"})
		return
	}

	rule := models.UserStockRule{
		UserID:         req.UserID,
		RuleName:       req.RuleName,
		RuleExpression: exprJSON,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := config.DB.Create(&rule).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "创建规则失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "规则创建成功", "rule_id": rule.ID})
}

// ======== 获取规则列表（分页） ========
type RuleData struct {
	ID             uint                   `json:"id"`
	UserID         string                 `json:"user_id"`
	RuleName       string                 `json:"rule_name"`
	RuleExpression map[string]interface{} `json:"rule_expression"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

type GetRulesResponse struct {
	Page     int        `json:"page" example:"1"`
	PageSize int        `json:"page_size" example:"20"`
	Total    int64      `json:"total" example:"100"`
	Data     []RuleData `json:"data"`
}

// GetRules 获取规则列表
// @Summary 获取用户选股规则（分页）
// @Description 根据 user_id 获取用户规则，可分页
// @Tags 用户选股规则
// @Produce json
// @Param user_id query string true "用户ID"
// @Param page query int false "页码，默认1"
// @Param page_size query int false "每页数量，默认20，最大100"
// @Success 200 {object} GetRulesResponse
// @Failure 400 {object} map[string]interface{}
// @Router /user/rules [get]
func GetRules(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 user_id"})
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

	query := config.DB.Model(&models.UserStockRule{}).Where("user_id = ?", userID)
	query.Count(&total)
	query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&rules)

	var resp []RuleData
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
		Page:     page,
		PageSize: pageSize,
		Total:    total,
		Data:     resp,
	})
}

// ======== 删除规则 ========
// DeleteRule 删除规则
// @Summary 删除选股规则
// @Description 根据规则ID删除用户规则
// @Tags 用户选股规则
// @Produce json
// @Param id path int true "规则ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /user/rules/{id} [delete]
func DeleteRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID无效"})
		return
	}

	if err := config.DB.Delete(&models.UserStockRule{}, id).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "规则删除成功"})
}

// ======== 更新规则 ========
type UpdateRuleRequest struct {
	RuleName       string                 `json:"rule_name" example:"突破均线"`
	RuleExpression map[string]interface{} `json:"rule_expression" example:"{\"ma5_gt_ma10\": true}"`
}

// UpdateRule 更新规则
// @Summary 更新选股规则
// @Description 根据规则ID更新用户规则
// @Tags 用户选股规则
// @Accept json
// @Produce json
// @Param id path int true "规则ID"
// @Param rule body UpdateRuleRequest true "更新内容"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /user/rules/{id} [put]
func UpdateRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID无效"})
		return
	}

	var req UpdateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var rule models.UserStockRule
	if err := config.DB.First(&rule, id).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "规则不存在"})
		return
	}

	if req.RuleName != "" {
		rule.RuleName = req.RuleName
	}
	if req.RuleExpression != nil {
		exprJSON, _ := json.Marshal(req.RuleExpression)
		rule.RuleExpression = exprJSON
	}

	rule.UpdatedAt = time.Now()
	if err := config.DB.Save(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	var expr map[string]interface{}
	_ = json.Unmarshal(rule.RuleExpression, &expr)

	c.JSON(http.StatusOK, gin.H{"message": "规则更新成功", "rule": RuleData{
		ID:             rule.ID,
		UserID:         rule.UserID,
		RuleName:       rule.RuleName,
		RuleExpression: expr,
		CreatedAt:      rule.CreatedAt,
		UpdatedAt:      rule.UpdatedAt,
	}})
}
