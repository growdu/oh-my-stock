package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"oh-my-stock/config"

	"github.com/gin-gonic/gin"
)

type ScreenRequest struct {
	Industry	string	`form:"industry"`
	Market	string	`form:"market"`
	Keyword	string	`form:"keyword"`
	PriceMin	*float64	`form:"price_min"`
	PriceMax	*float64	`form:"price_max"`
	ChangeMin	*float64	`form:"change_min"`
	ChangeMax	*float64	`form:"change_max"`
	VolumeMin	*float64	`form:"volume_min"`
	VolumeMax	*float64	`form:"volume_max"`
	TurnoverMin	*float64	`form:"turnover_min"`
	TurnoverMax	*float64	`form:"turnover_max"`
	PETTMMin	*float64	`form:"pe_ttm_min"`
	PETTMMax	*float64	`form:"pe_ttm_max"`
	PBMin	*float64	`form:"pb_min"`
	PBMax	*float64	`form:"pb_max"`
	NetAmountMin	*float64	`form:"net_amount_min"`
	NetAmountMax	*float64	`form:"net_amount_max"`
	SortBy	string	`form:"sort_by"`
	SortOrder	string	`form:"sort_order"`
	Page	int	`form:"page"`
	PageSize	int	`form:"page_size"`
}

type ScreenResult struct {
	Symbol	string	`json:"symbol"`
	Name	string	`json:"name"`
	Industry	string	`json:"industry"`
	Market	string	`json:"market"`
	Close	float64	`json:"close"`
	Open	float64	`json:"open"`
	High	float64	`json:"high"`
	Low	float64	`json:"low"`
	ChangePercent	float64	`json:"change_percent"`
	Volume	float64	`json:"volume"`
	TurnoverRate	float64	`json:"turnover_rate"`
	PETTM	float64	`json:"pe_ttm"`
	PB	float64	`json:"pb"`
	InflowAmount	float64	`json:"in_amount"`
	OutflowAmount	float64	`json:"out_amount"`
	NetAmount	float64	`json:"net_amount"`
	TradeDate	string	`json:"trade_date"`
}
// ScreenStocks filters stocks by multiple criteria
func ScreenStocks(c *gin.Context) {
	var req ScreenRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Page <= 0 { req.Page = 1 }
	if req.PageSize <= 0 || req.PageSize > 200 { req.PageSize = 20 }

	baseQuery := `SELECT 
		h.symbol, h.name,
		COALESCE(b.industry, '') AS industry,
		COALESCE(b.market, '') AS market,
		COALESCE(h.close, 0) AS close,
		COALESCE(h.open, 0) AS open,
		COALESCE(h.high, 0) AS high,
		COALESCE(h.low, 0) AS low,
		COALESCE(h.change_percent, 0) AS change_percent,
		COALESCE(h.volume, 0) AS volume,
		COALESCE(h.turnover_rate, 0) AS turnover_rate,
		COALESCE(b.pettm, 0) AS pe_ttm,
		COALESCE(b.pb, 0) AS pb,
		COALESCE(h.in_amount, 0) AS inflow_amount,
		COALESCE(h.out_amount, 0) AS outflow_amount,
		COALESCE(h.net_amount, 0) AS net_amount,
		TO_CHAR(h.trade_date, 'YYYY-MM-DD') AS trade_date
	FROM stock_history_mv h
	LEFT JOIN stock_basic_info b ON h.symbol = b.symbol
	LEFT JOIN stock_daily_data d ON h.symbol = d.symbol AND h.trade_date = d.trade_date`

	var conditions []string
	var args []interface{}
	argIdx := 1

	if req.Industry != "" {
		conditions = append(conditions, fmt.Sprintf("b.industry = $%d", argIdx))
		args = append(args, req.Industry)
		argIdx++
	}
	if req.Market != "" {
		conditions = append(conditions, fmt.Sprintf("b.market = $%d", argIdx))
		args = append(args, req.Market)
		argIdx++
	}
	if req.Keyword != "" {
		conditions = append(conditions, fmt.Sprintf("(h.symbol LIKE $%d OR h.name LIKE $%d)", argIdx, argIdx+1))
		args = append(args, "%"+req.Keyword+"%", "%"+req.Keyword+"%")
		argIdx += 2
	}
	if req.PriceMin != nil {
		conditions = append(conditions, fmt.Sprintf("h.close >= $%d", argIdx))
		args = append(args, *req.PriceMin)
		argIdx++
	}
	if req.PriceMax != nil {
		conditions = append(conditions, fmt.Sprintf("h.close <= $%d", argIdx))
		args = append(args, *req.PriceMax)
		argIdx++
	}
	if req.ChangeMin != nil {
		conditions = append(conditions, fmt.Sprintf("h.change_percent >= $%d", argIdx))
		args = append(args, *req.ChangeMin)
		argIdx++
	}
	if req.ChangeMax != nil {
		conditions = append(conditions, fmt.Sprintf("h.change_percent <= $%d", argIdx))
		args = append(args, *req.ChangeMax)
		argIdx++
	}
	if req.VolumeMin != nil {
		conditions = append(conditions, fmt.Sprintf("h.volume >= $%d", argIdx))
		args = append(args, *req.VolumeMin)
		argIdx++
	}
	if req.VolumeMax != nil {
		conditions = append(conditions, fmt.Sprintf("h.volume <= $%d", argIdx))
		args = append(args, *req.VolumeMax)
		argIdx++
	}
	if req.TurnoverMin != nil {
		conditions = append(conditions, fmt.Sprintf("h.turnover_rate >= $%d", argIdx))
		args = append(args, *req.TurnoverMin)
		argIdx++
	}
	if req.TurnoverMax != nil {
		conditions = append(conditions, fmt.Sprintf("h.turnover_rate <= $%d", argIdx))
		args = append(args, *req.TurnoverMax)
		argIdx++
	}
	if req.PETTMMin != nil {
		conditions = append(conditions, fmt.Sprintf("b.pettm >= $%d", argIdx))
		args = append(args, *req.PETTMMin)
		argIdx++
	}
	if req.PETTMMax != nil {
		conditions = append(conditions, fmt.Sprintf("b.pettm <= $%d", argIdx))
		args = append(args, *req.PETTMMax)
		argIdx++
	}
	if req.PBMin != nil {
		conditions = append(conditions, fmt.Sprintf("b.pb >= $%d", argIdx))
		args = append(args, *req.PBMin)
		argIdx++
	}
	if req.PBMax != nil {
		conditions = append(conditions, fmt.Sprintf("b.pb <= $%d", argIdx))
		args = append(args, *req.PBMax)
		argIdx++
	}
	if req.NetAmountMin != nil {
		conditions = append(conditions, fmt.Sprintf("h.net_amount >= $%d", argIdx))
		args = append(args, *req.NetAmountMin)
		argIdx++
	}
	if req.NetAmountMax != nil {
		conditions = append(conditions, fmt.Sprintf("h.net_amount <= $%d", argIdx))
		args = append(args, *req.NetAmountMax)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	sortBy := "h.change_percent"
	switch req.SortBy {
	case "volume": sortBy = "h.volume"
	case "turnover_rate": sortBy = "h.turnover_rate"
	case "pe_ttm": sortBy = "b.pettm"
	case "pb": sortBy = "b.pb"
	case "net_amount": sortBy = "h.net_amount"
	case "close": sortBy = "h.close"
	}
	sortOrder := "DESC"
	if strings.ToLower(req.SortOrder) == "asc" { sortOrder = "ASC" }

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM ("+baseQuery+" "+whereClause+") sub")
	var total int64
	config.DB.Raw(countQuery, args...).Scan(&total)

	offset := (req.Page - 1) * req.PageSize
	dataQuery := fmt.Sprintf(baseQuery+" "+whereClause+" ORDER BY "+sortBy+" "+sortOrder+" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	queryArgs := append(args, req.PageSize, offset)

	var results []ScreenResult
	if err := config.DB.Raw(dataQuery, queryArgs...).Scan(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if results == nil { results = []ScreenResult{} }

	c.JSON(http.StatusOK, gin.H{
		"page": req.Page,
		"page_size": req.PageSize,
		"total": total,
		"data": results,
	})
}
// GetIndustryList returns all distinct industries
func GetIndustryList(c *gin.Context) {
	var industries []string
	config.DB.Raw("SELECT DISTINCT industry FROM stock_basic_info WHERE industry != '' ORDER BY industry").Scan(&industries)
	if industries == nil { industries = []string{} }
	c.JSON(http.StatusOK, industries)
}

// GetMarketList returns all distinct market types
func GetMarketList(c *gin.Context) {
	var markets []string
	config.DB.Raw("SELECT DISTINCT market FROM stock_basic_info WHERE market != '' ORDER BY market").Scan(&markets)
	if markets == nil { markets = []string{} }
	c.JSON(http.StatusOK, markets)
}

// GetStockRanking returns stock rankings by different dimensions.
// query: rank_by=change_percent|volume|turnover_rate|net_amount,
//        order=asc|desc (default desc), limit=1..100 (default 20).
func GetStockRanking(c *gin.Context) {
	rankBy := c.DefaultQuery("rank_by", "change_percent")
	order := c.DefaultQuery("order", "desc")
	limitStr := c.DefaultQuery("limit", "20")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 100 { limit = 20 }

	orderField := "change_percent"
	switch rankBy {
	case "volume": orderField = "volume"
	case "turnover_rate": orderField = "turnover_rate"
	case "net_amount": orderField = "net_amount"
	}
	sortOrder := "DESC"
	if strings.ToLower(order) == "asc" {
		sortOrder = "ASC"
	}

	var results []ScreenResult
	query := fmt.Sprintf(`SELECT h.symbol, h.name, COALESCE(b.industry, '') AS industry, COALESCE(b.market, '') AS market, COALESCE(h.close,0) AS close, COALESCE(h.change_percent,0) AS change_percent, COALESCE(h.volume,0) AS volume, COALESCE(h.turnover_rate,0) AS turnover_rate, COALESCE(h.net_amount,0) AS net_amount, TO_CHAR(h.trade_date,'YYYY-MM-DD') AS trade_date FROM stock_history_mv h LEFT JOIN stock_basic_info b ON h.symbol=b.symbol ORDER BY h.%s %s LIMIT %d`, orderField, sortOrder, limit)
	config.DB.Raw(query).Scan(&results)
	if results == nil { results = []ScreenResult{} }
	c.JSON(http.StatusOK, gin.H{"rank_by": rankBy, "order": strings.ToLower(order), "data": results})
}