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

// @Summary 创建指标记录
// @Tags 股票指标
// @Accept json
// @Produce json
// @Param data body models.StockIndicator true "股票指标数据"
// @Success 201 {object} models.StockIndicator
// @Failure 400 {string} string "Bad Request"
// @Router /stock-indicators [post]
func CreateStockIndicator(c *gin.Context) {
	var indicator models.StockIndicator
	if err := c.ShouldBindJSON(&indicator); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := config.DB.Create(&indicator).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, indicator)
}

// @Summary 获取全部指标
// @Tags 股票指标
// @Produce json
// @Success 200 {array} models.StockIndicator
// @Router /stock-indicators [get]
func GetStockIndicators(c *gin.Context) {
	var indicators []models.StockIndicator
	if err := config.DB.Find(&indicators).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, indicators)
}

// @Summary 根据ID获取指标
// @Tags 股票指标
// @Produce json
// @Param id path int true "指标ID"
// @Success 200 {object} models.StockIndicator
// @Failure 404 {string} string "Not Found"
// @Router /stock-indicators/{id} [get]
func GetStockIndicatorByID(c *gin.Context) {
	id := c.Param("id")
	var indicator models.StockIndicator
	if err := config.DB.First(&indicator, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, "Not Found")
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, indicator)
}

// @Summary 根据symbol和calc_date获取指标
// @Tags 股票指标
// @Produce json
// @Param symbol path string true "股票代码"
// @Param calc_date query string true "计算日期(YYYY-MM-DD)"
// @Success 200 {object} models.StockIndicator
// @Failure 400 {string} string "Bad Request"
// @Failure 404 {string} string "Not Found"
// @Router /stock-indicators/symbol/{symbol} [get]
func GetStockIndicatorBySymbolAndDate(c *gin.Context) {
	symbol := c.Param("symbol")
	dateStr := c.Query("calc_date")

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid calc_date"})
		return
	}

	var indicator models.StockIndicator
	if err := config.DB.Where("symbol = ? AND calc_date = ?", symbol, date).
		First(&indicator).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, "Not Found")
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, indicator)
}

// @Summary 更新指标
// @Tags 股票指标
// @Accept json
// @Produce json
// @Param id path int true "指标ID"
// @Param data body models.StockIndicator true "股票指标数据"
// @Success 200 {object} models.StockIndicator
// @Failure 400 {string} string "Bad Request"
// @Failure 404 {string} string "Not Found"
// @Router /stock-indicators/{id} [put]
func UpdateStockIndicator(c *gin.Context) {
	id := c.Param("id")
	var indicator models.StockIndicator
	if err := config.DB.First(&indicator, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, "Not Found")
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	var input models.StockIndicator
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config.DB.Model(&indicator).Updates(input)
	c.JSON(http.StatusOK, indicator)
}

// @Summary 删除指标
// @Tags 股票指标
// @Param id path int true "指标ID"
// @Success 204 {string} string "No Content"
// @Failure 404 {string} string "Not Found"
// @Router /stock-indicators/{id} [delete]
func DeleteStockIndicator(c *gin.Context) {
	id := c.Param("id")
	if err := config.DB.Delete(&models.StockIndicator{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
