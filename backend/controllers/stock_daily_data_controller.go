package controllers

import (
	"fmt"
	"context"
	"log"
	"net/http"
	"time"

	"oh-my-stock/config"
	"oh-my-stock/fetcher"
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

// GetRecentStockDaily 读取最近 7 天（DB 优先）
func GetRecentStockDaily(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol required"})
		return
	}

	// 1) 优先 DB
	var rows []models.StockDailyData
	if err := config.DB.Where("symbol = ?", symbol).
		Order("trade_date DESC").
		Limit(7).Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 2) 缓存判断：是否覆盖最近 7 个自然日（按 trade_date）
	if rowsCoverLastDays(rows, 7) {
		c.JSON(http.StatusOK, gin.H{"source": "db", "data": rows})
		return
	}

	// 3) DB 不够 → 联网抓 7 天
	if err := refreshOneSymbol(context.Background(), symbol, 7); err != nil {
		log.Printf("⚠️ refresh %s: %v", symbol, err)
		c.JSON(http.StatusOK, gin.H{"source": "db_partial", "data": rows, "refresh_error": err.Error()})
		return
	}
	if err := config.DB.Where("symbol = ?", symbol).
		Order("trade_date DESC").
		Limit(7).Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"source": "db", "data": rows})
}

// RefreshStockDaily 强制刷新单只股票最近 7 天日线
func RefreshStockDaily(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol required"})
		return
	}
	if err := refreshOneSymbol(context.Background(), symbol, 7); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var rows []models.StockDailyData
	if err := config.DB.Where("symbol = ?", symbol).
		Order("trade_date DESC").
		Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"source": "fresh", "data": rows})
}

func refreshOneSymbol(ctx context.Context, symbol string, days int) error {
	daily, err := fetcher.FetchRecentDaily(ctx, symbol, days)
	if err != nil {
		return err
	}
	if len(daily) == 0 {
		return nil
	}
	prepared := make([]models.StockDailyData, 0, len(daily))
	for i, r := range daily {
		t, perr := time.Parse("2006-01-02", r.Day)
		if perr != nil {
			continue
		}
		entry := models.StockDailyData{
			Symbol:    symbol,
			TradeDate: t,
			Open:      r.Open,
			High:      r.High,
			Low:       r.Low,
			Close:     r.Close,
			Volume:    int64(r.Volume),
			Turnover:  r.Turnover,
		}
		if i > 0 {
			prev := daily[i-1].Close
			if prev != 0 {
				entry.ChangeAmount = fetcher.Round4(r.Close - prev)
				entry.ChangePercent = fetcher.Round4((r.Close - prev) / prev * 100)
			}
		}
		prepared = append(prepared, entry)
	}
	if n, err := fetcher.UpsertDaily(prepared); err != nil {
		return fmt.Errorf("upsert daily: %w", err)
	} else {
		log.Printf("✅ %s 写入 daily %d 行", symbol, n)
	}

	// 资金流（best-effort）
	flows, ferr := fetcher.FetchEastMoneyFlowDays(ctx, symbol, days)
	if ferr != nil {
		log.Printf("⚠️ %s 拉资金流失败: %v", symbol, ferr)
	} else if n, ferr := fetcher.UpsertMoneyFlowDaily(flows); ferr != nil {
		log.Printf("⚠️ %s 写资金流失败: %v", symbol, ferr)
	} else {
		log.Printf("✅ %s 写入资金流 %d 行", symbol, n)
	}

	if n, err := fetcher.UpsertHistoryMV(prepared); err != nil {
		log.Printf("⚠️ %s 同步 mv: %v", symbol, err)
	} else {
		log.Printf("✅ %s 写入 mv %d 行", symbol, n)
	}

	// 顺手裁剪
	if n, perr := fetcher.PurgeOldDaily(); perr == nil && n > 0 {
		log.Printf("✅ %s 触发后裁剪 stock_daily_data %d 行", symbol, n)
	}

	// 技术指标
	recent, err := fetcher.LoadRecentDaily(symbol, 90)
	if err == nil && len(recent) >= 5 {
		inds := fetcher.ComputeIndicators(symbol, recent)
		if n, err := fetcher.UpsertIndicators(inds); err != nil {
			log.Printf("⚠️ %s 写技术指标失败: %v", symbol, err)
		} else {
			log.Printf("✅ %s 写入指标 %d 行", symbol, n)
		}
	}
	return nil
}

// rowsCoverLastDays DB 行是否覆盖最近 n 天（按 trade_date 自然日）
func rowsCoverLastDays(rows []models.StockDailyData, n int) bool {
	if len(rows) == 0 {
		return false
	}
	now := time.Now()
	cutoff := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).
		AddDate(0, 0, -n+1)
	have := map[string]bool{}
	for _, r := range rows {
		have[r.TradeDate.Format("2006-01-02")] = true
	}
	// 至少覆盖 recent n 个日历日中已存在交易
	for d := time.Now(); !d.Before(cutoff); d = d.AddDate(0, 0, -1) {
		// 工作日跳过周末非交易日：实际不严格，遇周末可能不覆盖；放宽到 70% 即可
		weekday := d.Weekday()
		if weekday == time.Saturday || weekday == time.Sunday {
			continue
		}
		if !have[d.Format("2006-01-02")] {
			return false
		}
	}
	return true
}

