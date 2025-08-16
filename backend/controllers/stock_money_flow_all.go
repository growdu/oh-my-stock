package controllers

import (
	"errors"
	"net/http"
	"time"

	"oh-my-stock/config"
	"oh-my-stock/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// @Summary 创建资金流数据
// @Tags 股票资金流(ALL)
// @Accept json
// @Produce json
// @Param data body models.StockMoneyFlowAll true "股票资金流数据"
// @Success 201 {object} models.StockMoneyFlowAll
// @Failure 400 {string} string "Bad Request"
// @Router /stock-money-flow-all [post]
func CreateStockMoneyFlowAll(c *gin.Context) {
	var flow models.StockMoneyFlowAll
	if err := c.ShouldBindJSON(&flow); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := config.DB.Create(&flow).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, flow)
}

// @Summary 获取全部资金流数据
// @Tags 股票资金流(ALL)
// @Produce json
// @Success 200 {array} models.StockMoneyFlowAll
// @Router /stock-money-flow-all [get]
func GetStockMoneyFlowAlls(c *gin.Context) {
	var flows []models.StockMoneyFlowAll
	if err := config.DB.Find(&flows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, flows)
}

// @Summary 根据ID获取资金流数据
// @Tags 股票资金流(ALL)
// @Produce json
// @Param id path int true "ID"
// @Success 200 {object} models.StockMoneyFlowAll
// @Failure 404 {string} string "Not Found"
// @Router /stock-money-flow-all/{id} [get]
func GetStockMoneyFlowAllByID(c *gin.Context) {
	id := c.Param("id")
	var flow models.StockMoneyFlowAll
	if err := config.DB.First(&flow, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, "Not Found")
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, flow)
}

// @Summary 根据symbol和trade_date获取资金流数据
// @Tags 股票资金流(ALL)
// @Produce json
// @Param symbol path string true "股票代码"
// @Param trade_date query string false "交易日期(YYYY-MM-DD)"
// @Param time_span query int false "时间跨度(0,3,5,10)"
// @Success 200 {array} models.StockMoneyFlowAll
// @Failure 400 {string} string "Bad Request"
// @Failure 404 {string} string "Not Found"
// @Router /stock-money-flow-all/symbol/{symbol} [get]
func GetStockMoneyFlowAllBySymbolAndDate(c *gin.Context) {
	symbol := c.Param("symbol")
	dateStr := c.Query("trade_date")
	timeSpan := c.Query("time_span")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol must set"})
		return
	}
	query := config.DB.Where("symbol = ? ", symbol)
	if dateStr != "" {
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid trade_date"})
			return
		}
		query = query.Where("trade_date = ?", date)
	}
	if timeSpan != "" {
		query = query.Where("time_span = ?", timeSpan)
	}

	var flows []models.StockMoneyFlowAll
	if err := query.Find(&flows).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, "Not Found")
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, flows)
}

// @Summary 更新资金流数据
// @Tags 股票资金流(ALL)
// @Accept json
// @Produce json
// @Param id path int true "ID"
// @Param data body models.StockMoneyFlowAll true "股票资金流数据"
// @Success 200 {object} models.StockMoneyFlowAll
// @Failure 400 {string} string "Bad Request"
// @Failure 404 {string} string "Not Found"
// @Router /stock-money-flow-all/{id} [put]
func UpdateStockMoneyFlowAll(c *gin.Context) {
	id := c.Param("id")
	var flow models.StockMoneyFlowAll
	if err := config.DB.First(&flow, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, "Not Found")
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	var input models.StockMoneyFlowAll
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config.DB.Model(&flow).Updates(input)
	c.JSON(http.StatusOK, flow)
}

// @Summary 删除资金流数据
// @Tags 股票资金流(ALL)
// @Param id path int true "ID"
// @Success 204 {string} string "No Content"
// @Failure 404 {string} string "Not Found"
// @Router /stock-money-flow-all/{id} [delete]
func DeleteStockMoneyFlowAll(c *gin.Context) {
	id := c.Param("id")
	if err := config.DB.Delete(&models.StockMoneyFlowAll{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
