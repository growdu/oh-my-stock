package controllers

import (
	"net/http"

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
