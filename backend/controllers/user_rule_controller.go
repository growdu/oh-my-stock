package controllers

import (
	"net/http"
	"oh-my-stock/config"
	"oh-my-stock/models"
	"strconv"

	"github.com/gin-gonic/gin"
)

// @Summary 添加选股规则
// @Tags 用户选股规则
// @Accept json
// @Produce json
// @Param rule body models.RuleRequest true "规则信息"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /user/rules [post]
func AddRule(c *gin.Context) {
	var req struct {
		UserID         string                 `json:"user_id" binding:"required"`
		RuleName       string                 `json:"rule_name" binding:"required"`
		RuleExpression map[string]interface{} `json:"rule_expression" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rule := models.UserStockRule{
		UserID:         req.UserID,
		RuleName:       req.RuleName,
		RuleExpression: req.RuleExpression,
	}

	if err := config.DB.Create(&rule).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "创建规则失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "规则创建成功", "rule_id": rule.ID})
}

// @Summary 获取用户选股规则（分页）
// @Tags 用户选股规则
// @Produce json
// @Param user_id query string true "用户ID"
// @Param page query int false "页码，默认1"
// @Param page_size query int false "每页数量，默认20"
// @Success 200 {object} map[string]interface{}
// @Router /user/rules [get]
func GetRules(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 user_id"})
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
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

	c.JSON(http.StatusOK, gin.H{
		"page":      page,
		"page_size": pageSize,
		"total":     total,
		"data":      rules,
	})
}

// @Summary 删除选股规则
// @Tags 用户选股规则
// @Produce json
// @Param id path int true "规则ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /user/rules/{id} [delete]
func DeleteRule(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
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

// @Summary 更新选股规则
// @Tags 用户选股规则
// @Accept json
// @Produce json
// @Param id path int true "规则ID"
// @Param rule body models.RuleUpdateRequest true "更新内容"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /user/rules/{id} [put]
func UpdateRule(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID无效"})
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
	if err := config.DB.First(&rule, id).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "规则不存在"})
		return
	}

	if req.RuleName != "" {
		rule.RuleName = req.RuleName
	}
	if req.RuleExpression != nil {
		rule.RuleExpression = req.RuleExpression
	}

	if err := config.DB.Save(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "规则更新成功", "rule": rule})
}
