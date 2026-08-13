package presets

import (
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
)

// RunResult 单条命中。
type RunResult struct {
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name"`
	Industry      string  `json:"industry"`
	Market        string  `json:"market"`
	Open          float64 `json:"open"`
	Close         float64 `json:"close"`
	High          float64 `json:"high"`
	Low           float64 `json:"low"`
	ChangePercent float64 `json:"change_percent"`
	Volume        float64 `json:"volume"`
	TurnoverRate  float64 `json:"turnover_rate"`
	NetAmount     float64 `json:"net_amount"`
	PETTM         float64 `json:"pe_ttm"`
	PB            float64 `json:"pb"`
	TradeDate     string  `json:"trade_date"`
	BoardPriority int     `json:"board_priority"`

	// 技术指标快照：让前端卡片直接看到 MA/MACD/KDJ/RSI/BOLL 的当前值
	// 与最近 1 日的 lag 对比，便于做"金叉/死叉"高亮
	MA5       float64 `json:"ma5"`
	MA10      float64 `json:"ma10"`
	MA20      float64 `json:"ma20"`
	MA60      float64 `json:"ma60"`
	MA5Prev   float64 `json:"ma5_prev"`
	MA10Prev  float64 `json:"ma10_prev"`
	MA20Prev  float64 `json:"ma20_prev"`
	MACD      float64 `json:"macd"`
	DIF       float64 `json:"dif"`
	DEA       float64 `json:"dea"`
	K         float64 `json:"k"`
	D         float64 `json:"d"`
	J         float64 `json:"j"`
	RSI6      float64 `json:"rsi6"`
	RSI12     float64 `json:"rsi12"`
	RSI24     float64 `json:"rsi24"`
	BollUpper float64 `json:"boll_upper"`
	BollMid   float64 `json:"boll_mid"`
	BollLower float64 `json:"boll_lower"`
}

// Run 在 stock_history_mv 上执行预设规则表达式。
// 最新交易日为基准，向前回溯 30 天用于窗口谓词；上市新股用 basic_info 判断。
//
// expression 取 Preset.Expression，page/pageSize 简单分页。
func Run(db *gorm.DB, expression map[string]interface{}, page, pageSize int) ([]RunResult, int64, error) {
	exprJSON, err := marshalExpr(expression)
	if err != nil {
		return nil, 0, err
	}
	compiled, err := Compile(exprJSON)
	if err != nil {
		return nil, 0, err
	}

	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 50
	}

	const baseSQL = `
WITH latest_dt AS (
  SELECT MAX(trade_date) AS d FROM stock_history_mv
),
ranked AS (
  SELECT
    h.symbol,
    h.name,
    h.trade_date,
    h.open,
    h.close,
    h.high,
    h.low,
    h.volume,
    h.change_percent,
    h.turnover_rate,
    h.net_amount,
    b.industry,
    b.market,
    h.pe_ttm,
    h.pb,
    b.listing_date,
    b.outstanding_shares,
    b.total_shares,
    b.status,
    i.ma5, i.ma10, i.ma20, i.ma60,
    i.macd, i.dif, i.dea,
    i.rsi6, i.rsi12, i.rsi24,
    i.k, i.d, i.j,
    i.boll_upper, i.boll_mid, i.boll_lower,
    i.ma5_lag1, i.ma5_lag2, i.ma5_lag3,
    i.ma10_lag1, i.ma10_lag2, i.ma10_lag3,
    i.ma20_lag1, i.ma20_lag2, i.ma20_lag3,
    i.ma60_lag1,
    i.dif_lag1, i.dea_lag1,
    i.k_lag1, i.d_lag1,
    i.yang_lag0, i.yang_lag1, i.yang_lag2, i.yang_lag3,
    i.close_lag1, i.close_lag2, i.close_lag3, i.close_lag5,
    i.vol_lag1, i.vol_lag2, i.vol_lag3, i.vol_lag5,
    i.net_lag1, i.net_lag2, i.net_lag3,
    i.vol_avg5,
    i.low_min5, i.low_min20, i.low_min60,
    i.high_max5, i.high_max30, i.high_max60, i.high_max90
  FROM stock_history_mv h
  CROSS JOIN latest_dt
  LEFT JOIN stock_basic_info b ON b.symbol = h.symbol
  LEFT JOIN stock_indicators  i ON i.symbol = h.symbol AND i.calc_date = h.trade_date
  WHERE h.trade_date = latest_dt.d
),
latest AS (
  SELECT *
  FROM ranked r
  WHERE r.trade_date = (SELECT MAX(trade_date) FROM ranked)
)
SELECT
  latest.symbol, latest.name, latest.industry, latest.market,
  latest.open, latest.close, latest.high, latest.low,
  latest.change_percent, latest.volume, latest.turnover_rate, latest.net_amount,
  latest.pe_ttm AS pe_ttm, latest.pb,
  TO_CHAR(latest.trade_date, 'YYYY-MM-DD') AS trade_date,
  CASE
    WHEN latest.symbol LIKE '300%%' OR latest.symbol LIKE '301%%' THEN 1  -- 创业板
    WHEN latest.symbol LIKE '688%%' THEN 2                                -- 科创板
    WHEN latest.symbol LIKE '60%%' OR latest.symbol LIKE '00%%' OR latest.symbol LIKE '20%%' THEN 3  -- 主板
    ELSE 4
  END AS board_priority,
  COALESCE(latest.ma5, 0)       AS ma5,
  COALESCE(latest.ma10, 0)      AS ma10,
  COALESCE(latest.ma20, 0)      AS ma20,
  COALESCE(latest.ma60, 0)      AS ma60,
  COALESCE(latest.ma5_lag1, 0)  AS ma5_prev,
  COALESCE(latest.ma10_lag1, 0) AS ma10_prev,
  COALESCE(latest.ma20_lag1, 0) AS ma20_prev,
  COALESCE(latest.macd, 0)      AS macd,
  COALESCE(latest.dif, 0)       AS dif,
  COALESCE(latest.dea, 0)       AS dea,
  COALESCE(latest.k, 0)         AS k,
  COALESCE(latest.d, 0)         AS d,
  COALESCE(latest.j, 0)         AS j,
  COALESCE(latest.rsi6, 0)      AS rsi6,
  COALESCE(latest.rsi12, 0)     AS rsi12,
  COALESCE(latest.rsi24, 0)     AS rsi24,
  COALESCE(latest.boll_upper, 0) AS boll_upper,
  COALESCE(latest.boll_mid, 0)   AS boll_mid,
  COALESCE(latest.boll_lower, 0) AS boll_lower
FROM latest
LEFT JOIN stock_basic_info basic ON basic.symbol = latest.symbol
WHERE ` + "1=1" + `
  AND %s
ORDER BY board_priority ASC, latest.change_percent DESC, latest.symbol ASC
LIMIT %d OFFSET %d`

	q := fmt.Sprintf(baseSQL, compiled.Where, pageSize, (page-1)*pageSize)

	// count 走相同的 WHERE
	countSQL := fmt.Sprintf(`
WITH latest_dt AS (
  SELECT MAX(trade_date) AS d FROM stock_history_mv
),
ranked AS (
  SELECT
    h.symbol,
    h.name,
    h.trade_date,
    h.open,
    h.close,
    h.high,
    h.low,
    h.volume,
    h.change_percent,
    h.turnover_rate,
    h.net_amount,
    b.industry,
    b.market,
    h.pe_ttm,
    h.pb,
    b.listing_date,
    b.outstanding_shares,
    b.total_shares,
    b.status,
    i.ma5, i.ma10, i.ma20, i.ma60,
    i.macd, i.dif, i.dea,
    i.rsi6, i.rsi12, i.rsi24,
    i.k, i.d, i.j,
    i.boll_upper, i.boll_mid, i.boll_lower,
    i.ma5_lag1, i.ma5_lag2, i.ma5_lag3,
    i.ma10_lag1, i.ma10_lag2, i.ma10_lag3,
    i.ma20_lag1, i.ma20_lag2, i.ma20_lag3,
    i.ma60_lag1,
    i.dif_lag1, i.dea_lag1,
    i.k_lag1, i.d_lag1,
    i.yang_lag0, i.yang_lag1, i.yang_lag2, i.yang_lag3,
    i.close_lag1, i.close_lag2, i.close_lag3, i.close_lag5,
    i.vol_lag1, i.vol_lag2, i.vol_lag3, i.vol_lag5,
    i.net_lag1, i.net_lag2, i.net_lag3,
    i.vol_avg5,
    i.low_min5, i.low_min20, i.low_min60,
    i.high_max5, i.high_max30, i.high_max60, i.high_max90
  FROM stock_history_mv h
  CROSS JOIN latest_dt
  LEFT JOIN stock_basic_info b ON b.symbol = h.symbol
  LEFT JOIN stock_indicators  i ON i.symbol = h.symbol AND i.calc_date = h.trade_date
  WHERE h.trade_date = latest_dt.d
),
latest AS (
  SELECT * FROM ranked r
  WHERE r.trade_date = (SELECT MAX(trade_date) FROM ranked)
)
SELECT COUNT(*) FROM latest
LEFT JOIN stock_basic_info basic ON basic.symbol = latest.symbol
WHERE `+"1=1"+`
  AND %s`, compiled.Where)

	var total int64
	if err := db.Raw(countSQL, compiled.Args...).Scan(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count: %w", err)
	}

	var rows []RunResult
	if err := db.Raw(q, compiled.Args...).Scan(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("query: %w", err)
	}
	if rows == nil {
		rows = []RunResult{}
	}
	return rows, total, nil
}

func marshalExpr(e map[string]interface{}) ([]byte, error) {
	if e == nil {
		return nil, fmt.Errorf("nil expression")
	}
	return json.Marshal(e)
}
