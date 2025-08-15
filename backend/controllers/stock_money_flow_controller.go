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
// @Tags 股票资金流
// @Accept json
// @Produce json
// @Param data body models.StockMoneyFlow true "股票资金流数据"
// @Success 201 {object} models.StockMoneyFlow
// @Failure 400 {string} string "Bad Request"
// @Router /stock-money-flow [post]
func CreateStockMoneyFlow(c *gin.Context) {
	var flow models.StockMoneyFlow
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
// @Tags 股票资金流
// @Produce json
// @Success 200 {array} models.StockMoneyFlow
// @Router /stock-money-flow [get]
func GetStockMoneyFlows(c *gin.Context) {
	var flows []models.StockMoneyFlow
	if err := config.DB.Find(&flows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, flows)
}

// @Summary 根据ID获取资金流数据
// @Tags 股票资金流
// @Produce json
// @Param id path int true "ID"
// @Success 200 {object} models.StockMoneyFlow
// @Failure 404 {string} string "Not Found"
// @Router /stock-money-flow/{id} [get]
func GetStockMoneyFlowByID(c *gin.Context) {
	id := c.Param("id")
	var flow models.StockMoneyFlow
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
// @Tags 股票资金流
// @Produce json
// @Param symbol path string true "股票代码"
// @Param trade_date query string true "交易日期(YYYY-MM-DD)"
// @Success 200 {object} models.StockMoneyFlow
// @Failure 400 {string} string "Bad Request"
// @Failure 404 {string} string "Not Found"
// @Router /stock-money-flow/symbol/{symbol} [get]
func GetStockMoneyFlowBySymbolAndDate(c *gin.Context) {
	symbol := c.Param("symbol")
	dateStr := c.Query("trade_date")

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid trade_date"})
		return
	}

	var flow models.StockMoneyFlow
	if err := config.DB.Where("symbol = ? AND trade_date = ?", symbol, date).
		First(&flow).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, "Not Found")
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, flow)
}

// @Summary 更新资金流数据
// @Tags 股票资金流
// @Accept json
// @Produce json
// @Param id path int true "ID"
// @Param data body models.StockMoneyFlow true "股票资金流数据"
// @Success 200 {object} models.StockMoneyFlow
// @Failure 400 {string} string "Bad Request"
// @Failure 404 {string} string "Not Found"
// @Router /stock-money-flow/{id} [put]
func UpdateStockMoneyFlowV1(c *gin.Context) {
	id := c.Param("id")
	var flow models.StockMoneyFlow
	if err := config.DB.First(&flow, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, "Not Found")
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	var input models.StockMoneyFlow
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config.DB.Model(&flow).Updates(input)
	c.JSON(http.StatusOK, flow)
}

// @Summary 删除资金流数据
// @Tags 股票资金流
// @Param id path int true "ID"
// @Success 204 {string} string "No Content"
// @Failure 404 {string} string "Not Found"
// @Router /stock-money-flow/{id} [delete]
func DeleteStockMoneyFlowV1(c *gin.Context) {
	id := c.Param("id")
	if err := config.DB.Delete(&models.StockMoneyFlow{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
