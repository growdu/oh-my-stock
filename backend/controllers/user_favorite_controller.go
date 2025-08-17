package controllers

import (
	"net/http"
	"oh-my-stock/config"
	"oh-my-stock/models"
	"strconv"

	"github.com/gin-gonic/gin"
)

// @Summary 添加收藏股票
// @Tags 用户收藏
// @Accept json
// @Produce json
// @Param favorite body models.FavoriteRequest true "收藏信息"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /user/favorites [post]
func AddFavorite(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id" binding:"required"`
		Symbol string `json:"symbol" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	favorite := models.UserFavoriteStock{
		UserID: req.UserID,
		Symbol: req.Symbol,
	}

	if err := config.DB.Create(&favorite).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "添加收藏失败，可能已存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "收藏成功", "favorite_id": favorite.ID})
}

// @Summary 获取用户收藏的股票（分页）
// @Tags 用户收藏
// @Produce json
// @Param user_id query string true "用户ID"
// @Param page query int false "页码，默认1"
// @Param page_size query int false "每页数量，默认20"
// @Success 200 {object} map[string]interface{}
// @Router /user/favorites [get]
func GetFavorites(c *gin.Context) {
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
	var favorites []models.UserFavoriteStock

	query := config.DB.Model(&models.UserFavoriteStock{}).Where("user_id = ?", userID)
	query.Count(&total)

	query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&favorites)

	c.JSON(http.StatusOK, gin.H{
		"page":      page,
		"page_size": pageSize,
		"total":     total,
		"data":      favorites,
	})
}

// @Summary 删除收藏股票
// @Tags 用户收藏
// @Produce json
// @Param id path int true "收藏ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /user/favorites/{id} [delete]
func DeleteFavorite(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID 无效"})
		return
	}

	if err := config.DB.Delete(&models.UserFavoriteStock{}, id).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
