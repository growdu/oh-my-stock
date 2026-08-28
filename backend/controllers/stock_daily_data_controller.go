package controllers

import (
	"math"
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
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


// @Summary 获取单只股票的K线数据（含MA5/MA10/MA20）
// @Tags 股票日线数据
// @Produce json
// @Param symbol path string true "股票代码"
// @Param days    query int    false "回溯天数，默认 90，最大 365"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {string} string "Bad Request"
// @Router /stock-daily-data/{symbol}/kline [get]
// sanitizeF 把 math.NaN() / Inf 替换为 0（防止 json 编码失败）
func sanitizeF(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return v
}

// sanitizeP 把 *float64 中的 NaN / Inf 替换为 nil
func sanitizeP(p *float64) *float64 {
	if p == nil {
		return nil
	}
	if math.IsNaN(*p) || math.IsInf(*p, 0) {
		return nil
	}
	return p
}

func GetStockKLine(c *gin.Context) {
	symbol := c.Param("symbol")
	daysStr := c.DefaultQuery("days", "90")

	days, err := strconv.Atoi(daysStr)
	if err != nil || days <= 0 || days > 365 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "days must be 1..365"})
		return
	}

	// 用一条 SQL JOIN 同时拿到日线和 MA，避免前端两次请求
	type KlineRow struct {
		TradeDate     time.Time `json:"trade_date"`
		Open          float64   `json:"open"`
		High          float64   `json:"high"`
		Low           float64   `json:"low"`
		Close         float64   `json:"close"`
		Volume        int64     `json:"volume"`
		ChangePercent float64   `json:"change_percent"`
		TurnoverRate  float64   `json:"turnover_rate"`
		Ma5           *float64  `json:"ma5"`
		Ma10          *float64  `json:"ma10"`
		Ma20          *float64  `json:"ma20"`
		Macd          *float64  `json:"macd"`
		Dif           *float64  `json:"dif"`
		Dea           *float64  `json:"dea"`
		KdjK          *float64  `json:"k"   gorm:"column:k"`
		KdjD          *float64  `json:"d"   gorm:"column:d"`
		KdjJ          *float64  `json:"j"   gorm:"column:j"`
		Rsi6          *float64  `json:"rsi6" gorm:"column:rsi6"`
	}
	var rows []KlineRow

	sql := `
		SELECT d.trade_date, d.open, d.high, d.low, d.close,
		       d.volume, d.change_percent, d.turnover_rate,
		       i.ma5, i.ma10, i.ma20, i.macd, i.dif, i.dea,
		       i.k, i.d, i.j, i.rsi6
		FROM stock_daily_data d
		LEFT JOIN stock_indicators i
		  ON i.symbol = d.symbol AND i.calc_date = d.trade_date
		WHERE d.symbol = ?
		  AND d.trade_date >= (CURRENT_DATE - (? || ' days')::interval)
		ORDER BY d.trade_date ASC
	`
	if err := config.DB.Raw(sql, symbol, strconv.Itoa(days)).Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 顺便算一个 5-day MA 的合成线（日线图上叠加"近五日均价线"）— 用 close 五日均线
	type Candle struct {
		Date          string  `json:"date"`
		Open          float64 `json:"open"`
		High          float64 `json:"high"`
		Low           float64 `json:"low"`
		Close         float64 `json:"close"`
		Volume        int64   `json:"volume"`
		ChangePercent float64 `json:"change_percent"`
		TurnoverRate  float64 `json:"turnover_rate"`
		Ma5           float64 `json:"ma5"`
		Ma10          float64 `json:"ma10"`
		Ma20          float64 `json:"ma20"`
		Avg5          float64 `json:"avg5"`           // 5-day 平均价
		Macd          float64 `json:"macd"`           // MACD 柱
		Dif           float64 `json:"dif"`            // DIF 快线
		Dea           float64 `json:"dea"`            // DEA 慢线
		KdjK          float64 `json:"k"`
		KdjD          float64 `json:"d"`
		KdjJ          float64 `json:"j"`
		Rsi6          float64 `json:"rsi6"`
	}
	candles := make([]Candle, 0, len(rows))
	closes := make([]float64, 0, len(rows))
	for _, r := range rows {
		closes = append(closes, sanitizeF(r.Close))
	}
	for idx, r := range rows {
		var avg5 float64
		if idx >= 4 {
			sum := closes[idx] + closes[idx-1] + closes[idx-2] + closes[idx-3] + closes[idx-4]
			avg5 = sum / 5
		}
		c := Candle{
			Date:          r.TradeDate.Format("2006-01-02"),
			Open:          r.Open,
			High:          r.High,
			Low:           r.Low,
			Close:         r.Close,
			Volume:        r.Volume,
			ChangePercent: r.ChangePercent,
			TurnoverRate:  r.TurnoverRate,
			Avg5:          avg5,
		}
		if v := sanitizeP(r.Ma5);  v != nil { c.Ma5  = *v }
		if v := sanitizeP(r.Ma10); v != nil { c.Ma10 = *v }
		if v := sanitizeP(r.Ma20); v != nil { c.Ma20 = *v }
		if v := sanitizeP(r.Macd); v != nil { c.Macd = *v }
		if v := sanitizeP(r.Dif);  v != nil { c.Dif  = *v }
		if v := sanitizeP(r.Dea);  v != nil { c.Dea  = *v }
		if v := sanitizeP(r.KdjK); v != nil { c.KdjK = *v }
		if v := sanitizeP(r.KdjD); v != nil { c.KdjD = *v }
		if v := sanitizeP(r.KdjJ); v != nil { c.KdjJ = *v }
		if v := sanitizeP(r.Rsi6); v != nil { c.Rsi6 = *v }
		candles = append(candles, c)
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol":  symbol,
		"days":    days,
		"count":   len(candles),
		"candles": candles,
	})
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

