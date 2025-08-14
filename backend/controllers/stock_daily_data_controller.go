package controllers

import (
	"net/http"
	"time"

	"oh-my-stock/config"
	"oh-my-stock/models"

	"github.com/gin-gonic/gin"
)

// @Summary 获取全部股票日线数据
// @Tags 股票日线数据
// @Produce json
// @Success 200 {array} models.StockDailyData
// @Router /stock-daily-data [get]
func GetAllStockDailyData(c *gin.Context) {
	var data []models.StockDailyData
	config.DB.Find(&data)
	c.JSON(http.StatusOK, data)
}

// @Summary 根据股票代码和交易日期查询
// @Tags 股票日线数据
// @Produce json
// @Param symbol path string true "股票代码"
// @Param trade_date query string false "交易日期(YYYY-MM-DD)"
// @Success 200 {array} models.StockDailyData
// @Failure 404 {string} string "Not Found"
// @Router /stock-daily-data/{symbol} [get]
func GetStockDailyData(c *gin.Context) {
	symbol := c.Param("symbol")
	tradeDateStr := c.Query("trade_date")

	var records []models.StockDailyData
	query := config.DB.Where("symbol = ?", symbol)

	if tradeDateStr != "" {
		tradeDate, err := time.Parse("2006-01-02", tradeDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid trade_date"})
			return
		}
		query = query.Where("trade_date = ?", tradeDate)
	}

	if err := query.Find(&records).Error; err != nil || len(records) == 0 {
		c.JSON(http.StatusNotFound, "Not Found")
		return
	}

	c.JSON(http.StatusOK, records)
}

// @Summary 新增股票日线数据
// @Tags 股票日线数据
// @Accept json
// @Produce json
// @Param data body models.StockDailyData true "股票日线数据"
// @Success 201 {object} models.StockDailyData
// @Router /stock-daily-data [post]
func CreateStockDailyData(c *gin.Context) {
	var input models.StockDailyData
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config.DB.Create(&input)
	c.JSON(http.StatusCreated, input)
}

// @Summary 删除@Tags 股票日线数据
// @Tags 股票日线数据
// @Produce json
// @Param symbol path string true "股票代码"
// @Param trade_date query string true "交易日期(YYYY-MM-DD)"
// @Success 200 {string} string "Deleted"
// @Router /stock-daily-data/{symbol} [delete]
func DeleteStockDailyData(c *gin.Context) {
	symbol := c.Param("symbol")
	tradeDateStr := c.Query("trade_date")
	tradeDate, err := time.Parse("2006-01-02", tradeDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid trade_date"})
		return
	}

	if err := config.DB.Where("symbol = ? AND trade_date = ?", symbol, tradeDate).Delete(&models.StockDailyData{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Delete failed"})
		return
	}
	c.JSON(http.StatusOK, "Deleted")
}
