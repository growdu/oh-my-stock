package controllers

import (
	"net/http"
	"strconv"

	"oh-my-stock/config"
	"oh-my-stock/models"

	"github.com/gin-gonic/gin"
)

// @Summary 根据股票代码或名称查询股票基本信息
// @Tags 股票综合信息
// @Produce json
// @Param symbol query string true "股票代码或股票名称"
// @Success 200 {object} models.StockHistory
// @Failure 400 {string} string "Bad Request"
// @Failure 404 {string} string "Not Found"
// @Router /stocks/info [get]
func GetStockHistoryInfo(c *gin.Context) {
	symbolOrName := c.Query("symbol")
	if symbolOrName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol required"})
		return
	}

	var stock models.StockHistory
	if err := config.DB.Where("symbol = ? OR name = ?", symbolOrName, symbolOrName).
		First(&stock).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "stock not found"})
		return
	}

	c.JSON(http.StatusOK, stock)
}

// @Summary 分页获取所有股票基本信息
// @Tags 股票综合信息
// @Produce json
// @Param page query int false "页码，默认1"
// @Param page_size query int false "每页数量，默认20"
// @Success 200 {object} map[string]interface{}
// @Router /stocks/list [get]
func GetStockList(c *gin.Context) {
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
	var stocks []models.StockHistory

	query := config.DB.Model(&models.StockHistory{})
	query.Count(&total)

	query.Order("symbol").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&stocks)

	c.JSON(http.StatusOK, gin.H{
		"page":      page,
		"page_size": pageSize,
		"total":     total,
		"data":      stocks,
	})
}


// @Summary 分页获取热门股票（涨幅超过指定阈值，默认5%）
// @Tags 股票综合信息
// @Produce json
// @Param page query int false "页码，默认1"
// @Param page_size query int false "每页数量，默认20"
// @Param threshold query float64 false "涨幅阈值，默认5"
// @Success 200 {object} map[string]interface{}
// @Router /stocks/hot [get]
func GetHotStocks(c *gin.Context) {
    pageStr := c.DefaultQuery("page", "1")
    pageSizeStr := c.DefaultQuery("page_size", "20")
    thresholdStr := c.DefaultQuery("threshold", "5")

    page, _ := strconv.Atoi(pageStr)
    pageSize, _ := strconv.Atoi(pageSizeStr)
    threshold, _ := strconv.ParseFloat(thresholdStr, 64)

    if page <= 0 {
        page = 1
    }
    if pageSize <= 0 || pageSize > 100 {
        pageSize = 20
    }
    if threshold <= 0 {
        threshold = 5
    }

    var total int64
    var stocks []models.StockHistory

    query := config.DB.Model(&models.StockHistory{}).
        Where("change_percent > ?", threshold)

    query.Count(&total)

    query.Order("change_percent DESC").
        Offset((page - 1) * pageSize).
        Limit(pageSize).
        Find(&stocks)

    c.JSON(http.StatusOK, gin.H{
        "page":      page,
        "page_size": pageSize,
        "threshold": threshold,
        "total":     total,
        "data":      stocks,
    })
}
