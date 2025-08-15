package controllers

import (
	"net/http"
	"strconv"

	"oh-my-stock/config"
	"oh-my-stock/models"

	"github.com/gin-gonic/gin"
)

// GetStocks 获取所有股票信息
// @Summary 获取所有股票信息
// @Description 获取所有股票的基础信息
// @Tags 股票信息
// @Produce json
// @Success 200 {array} models.StockBasicInfo
// @Router /stocks [get]
func GetStocks(c *gin.Context) {
	var stocks []models.StockBasicInfo
	config.DB.Find(&stocks)
	c.JSON(http.StatusOK, stocks)
}

// GetStockByID 根据ID获取股票信息
// @Summary 根据ID获取股票信息
// @Description 通过股票ID获取详细信息
// @Tags 股票信息
// @Produce json
// @Param id path int true "股票ID"
// @Success 200 {object} models.StockBasicInfo
// @Failure 404 {object} map[string]string
// @Router /stocks/{id} [get]
func GetStockByID(c *gin.Context) {
	id := c.Param("id")
	var stock models.StockBasicInfo
	if err := config.DB.First(&stock, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "股票不存在"})
		return
	}
	c.JSON(http.StatusOK, stock)
}

// CreateStock 创建股票信息
// @Summary 创建股票信息
// @Description 新增一条股票基础信息
// @Tags 股票信息
// @Accept json
// @Produce json
// @Param stock body models.StockBasicInfo true "股票信息"
// @Success 201 {object} models.StockBasicInfo
// @Failure 400 {object} map[string]string
// @Router /stocks [post]
func CreateStock(c *gin.Context) {
	var stock models.StockBasicInfo
	if err := c.ShouldBindJSON(&stock); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config.DB.Create(&stock)
	c.JSON(http.StatusCreated, stock)
}

// UpdateStock 更新股票信息
// @Summary 更新股票信息
// @Description 根据ID更新股票基础信息
// @Tags 股票信息
// @Accept json
// @Produce json
// @Param id path int true "股票ID"
// @Param stock body models.StockBasicInfo true "股票信息"
// @Success 200 {object} models.StockBasicInfo
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /stocks/{id} [put]
func UpdateStock(c *gin.Context) {
	id := c.Param("id")
	var stock models.StockBasicInfo
	if err := config.DB.First(&stock, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "股票不存在"})
		return
	}
	if err := c.ShouldBindJSON(&stock); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config.DB.Save(&stock)
	c.JSON(http.StatusOK, stock)
}

// DeleteStock 删除股票信息
// @Summary 删除股票信息
// @Description 根据ID删除股票记录
// @Tags 股票信息
// @Produce json
// @Param id path int true "股票ID"
// @Success 204
// @Failure 404 {object} map[string]string
// @Router /stocks/{id} [delete]
func DeleteStock(c *gin.Context) {
	id := c.Param("id")
	var stock models.StockBasicInfo
	if err := config.DB.First(&stock, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "股票不存在"})
		return
	}
	config.DB.Delete(&stock)
	c.Status(http.StatusNoContent)
}

// GetStockBySymbol 根据股票代码获取股票信息
// @Summary 根据股票代码获取股票信息
// @Description 通过股票代码获取股票基础信息
// @Tags 股票信息
// @Produce json
// @Param symbol path string true "股票代码"
// @Success 200 {object} models.StockBasicInfo
// @Failure 404 {object} map[string]string
// @Router /stocks/symbol/{symbol} [get]
func GetStockBySymbol(c *gin.Context) {
	symbol := c.Param("symbol")
	var stock models.StockBasicInfo
	if err := config.DB.Where("symbol = ?", symbol).First(&stock).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "股票不存在"})
		return
	}
	c.JSON(http.StatusOK, stock)
}

// DeleteStockBySymbol 根据股票代码删除股票信息
// @Summary 根据股票代码删除股票信息
// @Description 通过股票代码删除股票基础信息
// @Tags 股票信息
// @Produce json
// @Param symbol path string true "股票代码"
// @Success 204
// @Failure 404 {object} map[string]string
// @Router /stocks/symbol/{symbol} [delete]
func DeleteStockBySymbol(c *gin.Context) {
	symbol := c.Param("symbol")
	var stock models.StockBasicInfo
	if err := config.DB.Where("symbol = ?", symbol).First(&stock).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "股票不存在"})
		return
	}
	config.DB.Delete(&stock)
	c.Status(http.StatusNoContent)
}

// @Summary 获取股票最近 N 天历史数据（含技术指标和资金流）
// @Tags 股票综合信息
// @Produce json
// @Param symbol query string true "股票代码"
// @Param days query int false "最近几天，默认7天"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {string} string "Bad Request"
// @Failure 404 {string} string "Not Found"
// @Router /stocks/history [get]
func GetStockHistory(c *gin.Context) {
	symbol := c.Query("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol required"})
		return
	}

	daysStr := c.DefaultQuery("days", "7")
	days, _ := strconv.Atoi(daysStr)
	if days <= 0 {
		days = 7
	}

	// 基本信息
	var basic models.StockBasicInfo
	if err := config.DB.Where("symbol = ?", symbol).First(&basic).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "stock not found"})
		return
	}

	// 最近 N 天日线数据
	var dailyData []models.StockDailyData
	config.DB.Where("symbol = ?", symbol).
		Order("trade_date DESC").
		Limit(days).Find(&dailyData)

	// 最近 N 天技术指标
	var indicators []models.StockIndicator
	config.DB.Where("symbol = ?", symbol).
		Order("calc_date DESC").
		Limit(days).Find(&indicators)

	// 最近 N 天资金流
	var moneyFlows []models.StockMoneyFlow
	config.DB.Where("symbol = ?", symbol).
		Order("trade_date DESC").
		Limit(days).Find(&moneyFlows)

	// 整合每日数据
	history := make([]map[string]interface{}, 0, days)
	for i := 0; i < len(dailyData); i++ {
		d := dailyData[i]
		var ind models.StockIndicator
		var mf models.StockMoneyFlow

		// 找到对应日期的指标和资金流
		for _, x := range indicators {
			if x.CalcDate.Equal(d.TradeDate) {
				ind = x
				break
			}
		}
		for _, x := range moneyFlows {
			if x.TradeDate.Equal(d.TradeDate) {
				mf = x
				break
			}
		}

		record := gin.H{
			"trade_date":         d.TradeDate,
			"open":               d.Open,
			"close":              d.Close,
			"high":               d.High,
			"low":                d.Low,
			"volume":             d.Volume,
			"turnover_rate":      d.TurnoverRate,
			"change_percent":     d.ChangePercent,
			"ma5":                ind.MA5,
			"ma10":               ind.MA10,
			"ma20":               ind.MA20,
			"ma60":               ind.MA60,
			"macd":               ind.MACD,
			"dif":                ind.DIF,
			"dea":                ind.DEA,
			"k":                  ind.K,
			"d":                  ind.D,
			"j":                  ind.J,
			"rsi6":               ind.RSI6,
			"rsi12":              ind.RSI12,
			"rsi24":              ind.RSI24,
			"boll_upper":         ind.BollUpper,
			"boll_mid":           ind.BollMid,
			"boll_lower":         ind.BollLower,
			"main_net":           mf.MainNet,
			"retail_net":         mf.RetailNet,
			"large_order_ratio":  mf.LargeOrderRatio,
			"medium_order_ratio": mf.MediumOrderRatio,
			"small_order_ratio":  mf.SmallOrderRatio,
		}
		history = append(history, record)
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol":       basic.Symbol,
		"name":         basic.Name,
		"industry":     basic.Industry,
		"market":       basic.Market,
		"listing_date": basic.ListingDate,
		"daily_data":   history,
	})
}
