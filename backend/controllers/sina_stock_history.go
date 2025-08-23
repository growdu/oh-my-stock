package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// StockHistoryItem 单日历史行情数据
type StockHistoryItem struct {
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume float64 `json:"volume"`
}

// GetSinaStockHistory godoc
// @Summary      获取新浪股票历史数据
// @Description  返回指定股票的历史K线数据
// @Tags         股票
// @Accept       json
// @Produce      json
// @Param        code    query   string  true  "股票代码，例如 sh600000"
// @Param        period  query   string  false "周期 day/week/month，默认 day"
// @Success      200  {array}   StockHistoryItem
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /sina_stock_history [get]
func GetSinaStockHistory(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing code parameter"})
		return
	}

	period := c.DefaultQuery("period", "day")
	_ = period

	// 这里使用新浪通用历史接口示例
	// 日线：http://quotes.sina.cn/cn/api/json.php/StockHistoryService.getDailyKLine?symbol=sh600000
	// 注意新浪历史数据接口可能需要解析 JSONP 或特定格式

	// 简单示例用新浪免费接口，返回类似CSV格式
	url := fmt.Sprintf("http://quotes.sina.cn/cn/api/json.php/StockHistoryService.getDailyKLine?symbol=%s", code)

	resp, err := http.Get(url)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	dataStr := string(body)

	// 新浪历史数据可能是 JSONP 格式，需要处理
	dataStr = strings.TrimPrefix(dataStr, "StockHistoryService.getDailyKLine(")
	dataStr = strings.TrimSuffix(dataStr, ");")

	var rawData []map[string]string
	if err := json.Unmarshal([]byte(dataStr), &rawData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse response"})
		return
	}

	result := []StockHistoryItem{}
	for _, item := range rawData {
		open, _ := strconv.ParseFloat(item["open"], 64)
		high, _ := strconv.ParseFloat(item["high"], 64)
		low, _ := strconv.ParseFloat(item["low"], 64)
		closePrice, _ := strconv.ParseFloat(item["close"], 64)
		volume, _ := strconv.ParseFloat(item["volume"], 64)
		result = append(result, StockHistoryItem{
			Date:   item["date"],
			Open:   open,
			High:   high,
			Low:    low,
			Close:  closePrice,
			Volume: volume,
		})
	}

	c.JSON(http.StatusOK, result)
}

// GetSinaStocks godoc
// @Summary      获取新浪股票行情数据
// @Description  代理新浪接口，解决跨域问题，并自动将 GBK 转 UTF-8
// @Tags         股票
// @Accept       json
// @Produce      plain
// @Param        list  query   string  true  "股票代码列表（逗号分隔，例如：sh600000,sz000001,sz300750）"
// @Success      200   {string}  string  "返回原始行情字符串，UTF-8 编码"
// @Failure      400   {object}  map[string]string  "请求参数错误"
// @Failure      500   {object}  map[string]string  "服务器错误"
// @Router       /sina_stocks [get]
func GetSinaStocks(c *gin.Context) {
	list := c.Query("list")
	if list == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing list param"})
		return
	}

	url := "https://hq.sinajs.cn/list=" + list

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Referer", "http://finance.sina.com.cn/")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	utf8Data, _, _ := transform.Bytes(simplifiedchinese.GBK.NewDecoder(), body)

	c.Data(http.StatusOK, "text/plain; charset=utf-8", utf8Data)
}
