package controllers

import (
	"net/http"
	"oh-my-stock/config"
	"oh-my-stock/middleware"
	"oh-my-stock/models"
	"strconv"

	"github.com/gin-gonic/gin"
)

// AddFavorite 添加自选股（从 ctx.user_id 读，不再依赖 body.user_id）
func AddFavorite(c *gin.Context) {
	uid := middleware.GetUserID(c)
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	var req struct {
		Symbol string `json:"symbol" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 校验 symbol 是否在 stock_basic_info 中存在，避免拼错/过期代码入库。
	var basic models.StockBasicInfo
	if err := config.DB.Select("symbol, name").Where("symbol = ?", req.Symbol).First(&basic).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol 不存在或未入库"})
		return
	}

	fav := models.UserFavoriteStock{UserID: uid, Symbol: req.Symbol}
	if err := config.DB.Create(&fav).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "添加收藏失败，可能已存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "收藏成功", "favorite_id": fav.ID, "symbol": basic.Symbol, "name": basic.Name})
}

// GetFavorites 分页获取当前用户的自选股
func GetFavorites(c *gin.Context) {
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
	var favorites []models.UserFavoriteStock
	q := config.DB.Model(&models.UserFavoriteStock{}).Where("user_id = ?", uid)
	q.Count(&total)
	q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&favorites)

	// 附加股票基本信息 (从 MV 或 basic_info)
	type Row struct {
		models.UserFavoriteStock
		Name          string  `json:"name"`
		Market        string  `json:"market"`
		Industry      string  `json:"industry"`
		ChangePercent float64 `json:"change_percent"`
		CurrentPrice  float64 `json:"current_price"`
	}
	rows := make([]Row, 0, len(favorites))
	for _, f := range favorites {
		var basic models.StockBasicInfo
		var daily models.StockDailyData
		config.DB.Where("symbol = ?", f.Symbol).First(&basic)
		config.DB.Where("symbol = ?", f.Symbol).Order("trade_date DESC").First(&daily)
		rows = append(rows, Row{
			UserFavoriteStock: f,
			Name:              basic.Name,
			Market:            basic.Market,
			Industry:          basic.Industry,
			ChangePercent:     daily.ChangePercent,
			CurrentPrice:      daily.Close,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"page":      page,
		"page_size": pageSize,
		"total":     total,
		"data":      rows,
	})
}

// DeleteFavorite 按主键 ID 删除
func DeleteFavorite(c *gin.Context) {
	uid := middleware.GetUserID(c)
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID 无效"})
		return
	}
	res := config.DB.Where("id = ? AND user_id = ?", id, uid).Delete(&models.UserFavoriteStock{})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "记录不存在或无权限"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已取消收藏"})
}

// DeleteFavoriteBySymbol 新接口：按 symbol 直接删（前端约定）
// DELETE /user/favorites/symbol/:symbol
func DeleteFavoriteBySymbol(c *gin.Context) {
	uid := middleware.GetUserID(c)
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	symbol := c.Param("symbol")
	res := config.DB.Where("user_id = ? AND symbol = ?", uid, symbol).
		Delete(&models.UserFavoriteStock{})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "自选股不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已取消收藏", "symbol": symbol})
}
