package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"oh-my-stock/config"
	"oh-my-stock/presets"
	"strconv"
)

func ListPresets(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"data": presets.All}) }

func RunPreset(c *gin.Context) {
	preset := presets.ByID(c.Param("id"))
	if preset == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预设不存在"})
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	rows, total, err := presets.Run(config.DB, preset.Expression, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"preset": preset, "page": page, "page_size": pageSize, "total": total, "data": rows})
}
