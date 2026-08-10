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
WITH ranked AS (
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
    b.pettm,
    b.pb,
    b.listing_date,
    b.outstanding_shares,
    b.total_shares,
    b.status,
    i.ma5, i.ma10, i.ma20, i.ma60,
    i.macd, i.dif, i.dea,
    i.rsi6, i.rsi12, i.rsi24,
    i.k, i.d, i.j,
    i.boll_upper, i.boll_mid, i.boll_lower,
    (h.close > h.open) AS yang_lag0,
    LAG(h.close > h.open, 1) OVER w AS yang_lag1,
    LAG(h.close > h.open, 2) OVER w AS yang_lag2,
    LAG(h.close > h.open, 3) OVER w AS yang_lag3,
    LAG(h.close, 1) OVER w AS close_lag1,
    LAG(h.close, 2) OVER w AS close_lag2,
    LAG(h.close, 3) OVER w AS close_lag3,
    LAG(h.close, 5) OVER w AS close_lag5,
    LAG(h.volume, 1) OVER w AS vol_lag1,
    LAG(h.volume, 2) OVER w AS vol_lag2,
    LAG(h.volume, 3) OVER w AS vol_lag3,
    LAG(h.volume, 5) OVER w AS vol_lag5,
    LAG(h.net_amount, 1) OVER w AS net_lag1,
    LAG(h.net_amount, 2) OVER w AS net_lag2,
    LAG(h.net_amount, 3) OVER w AS net_lag3,
    AVG(h.volume) OVER (PARTITION BY h.symbol ORDER BY h.trade_date ROWS BETWEEN 5 PRECEDING AND 1 PRECEDING) AS vol_avg5,
    MAX(h.high)    OVER (PARTITION BY h.symbol ORDER BY h.trade_date ROWS BETWEEN 5  PRECEDING AND 1 PRECEDING)  AS high_max5,
    MAX(h.high)    OVER (PARTITION BY h.symbol ORDER BY h.trade_date ROWS BETWEEN 30 PRECEDING AND 1 PRECEDING) AS high_max30,
    MAX(h.high)    OVER (PARTITION BY h.symbol ORDER BY h.trade_date ROWS BETWEEN 60 PRECEDING AND 1 PRECEDING) AS high_max60,
    MAX(h.high)    OVER (PARTITION BY h.symbol ORDER BY h.trade_date ROWS BETWEEN 90 PRECEDING AND 1 PRECEDING) AS high_max90,
    LAG(i.ma5,  2) OVER w AS ma5_lag2,
    LAG(i.ma10, 2) OVER w AS ma10_lag2,
    LAG(i.ma20, 2) OVER w AS ma20_lag2,
    LAG(i.ma60, 2) OVER w AS ma60_lag2,
    LAG(i.ma5,  3) OVER w AS ma5_lag3,
    LAG(i.ma10, 3) OVER w AS ma10_lag3,
    LAG(i.ma20, 3) OVER w AS ma20_lag3,
    LAG(i.ma60, 3) OVER w AS ma60_lag3,
    LAG(i.ma5,  4) OVER w AS ma5_lag4,
    LAG(i.ma10, 4) OVER w AS ma10_lag4,
    LAG(i.ma20, 4) OVER w AS ma20_lag4,
    LAG(i.ma60, 4) OVER w AS ma60_lag4,
    LAG(i.ma5,  5) OVER w AS ma5_lag5,
    LAG(i.ma10, 5) OVER w AS ma10_lag5,
    LAG(i.ma20, 5) OVER w AS ma20_lag5,
    LAG(i.ma60, 5) OVER w AS ma60_lag5,
    LAG(i.dif,  1) OVER w AS dif_lag1,
    LAG(i.dea,  1) OVER w AS dea_lag1,
    LAG(i.k,    1) OVER w AS k_lag1,
    LAG(i.d,    1) OVER w AS d_lag1
  FROM stock_history_mv h
  LEFT JOIN stock_basic_info b ON b.symbol = h.symbol
  LEFT JOIN stock_indicators  i ON i.symbol = h.symbol AND i.calc_date = h.trade_date
  WHERE h.trade_date >= (SELECT MAX(trade_date) FROM stock_history_mv) - INTERVAL '90 days'
  WINDOW w AS (PARTITION BY h.symbol ORDER BY h.trade_date)
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
  latest.pettm AS pe_ttm, latest.pb,
  TO_CHAR(latest.trade_date, 'YYYY-MM-DD') AS trade_date
FROM latest
LEFT JOIN stock_basic_info basic ON basic.symbol = latest.symbol
WHERE ` + "1=1" + `
  AND %s
ORDER BY latest.change_percent DESC, latest.symbol ASC
LIMIT %d OFFSET %d`

	q := fmt.Sprintf(baseSQL, compiled.Where, pageSize, (page-1)*pageSize)

	// count 走相同的 WHERE
	countSQL := fmt.Sprintf(`
WITH ranked AS (
  SELECT
    h.symbol, h.name, h.trade_date, h.close, h.open, h.volume, h.change_percent, h.turnover_rate, h.net_amount, h.high,
    b.pettm, b.pb, b.industry, b.market, b.listing_date, b.outstanding_shares, b.total_shares, b.status,
    i.ma5, i.ma10, i.ma20, i.ma60, i.macd, i.dif, i.dea, i.rsi6, i.rsi12, i.rsi24,
    i.k, i.d, i.j, i.boll_upper, i.boll_mid, i.boll_lower,
    (h.close > h.open) AS yang_lag0,
    LAG(h.close > h.open, 1) OVER w AS yang_lag1,
    LAG(h.close > h.open, 2) OVER w AS yang_lag2,
    LAG(h.close > h.open, 3) OVER w AS yang_lag3,
    LAG(h.close, 1) OVER w AS close_lag1,
    LAG(h.close, 2) OVER w AS close_lag2,
    LAG(h.close, 3) OVER w AS close_lag3,
    LAG(h.close, 5) OVER w AS close_lag5,
    LAG(h.volume, 1) OVER w AS vol_lag1,
    LAG(h.volume, 2) OVER w AS vol_lag2,
    LAG(h.volume, 3) OVER w AS vol_lag3,
    LAG(h.volume, 5) OVER w AS vol_lag5,
    LAG(h.net_amount, 1) OVER w AS net_lag1,
    LAG(h.net_amount, 2) OVER w AS net_lag2,
    LAG(h.net_amount, 3) OVER w AS net_lag3,
    AVG(h.volume) OVER (PARTITION BY h.symbol ORDER BY h.trade_date ROWS BETWEEN 5 PRECEDING AND 1 PRECEDING) AS vol_avg5,
    MAX(h.high)    OVER (PARTITION BY h.symbol ORDER BY h.trade_date ROWS BETWEEN 5  PRECEDING AND 1 PRECEDING)  AS high_max5,
    MAX(h.high)    OVER (PARTITION BY h.symbol ORDER BY h.trade_date ROWS BETWEEN 30 PRECEDING AND 1 PRECEDING) AS high_max30,
    MAX(h.high)    OVER (PARTITION BY h.symbol ORDER BY h.trade_date ROWS BETWEEN 60 PRECEDING AND 1 PRECEDING) AS high_max60,
    MAX(h.high)    OVER (PARTITION BY h.symbol ORDER BY h.trade_date ROWS BETWEEN 90 PRECEDING AND 1 PRECEDING) AS high_max90,
    LAG(i.ma5,  2) OVER w AS ma5_lag2,
    LAG(i.ma10, 2) OVER w AS ma10_lag2,
    LAG(i.ma20, 2) OVER w AS ma20_lag2,
    LAG(i.ma60, 2) OVER w AS ma60_lag2,
    LAG(i.ma5,  3) OVER w AS ma5_lag3,
    LAG(i.ma10, 3) OVER w AS ma10_lag3,
    LAG(i.ma20, 3) OVER w AS ma20_lag3,
    LAG(i.ma60, 3) OVER w AS ma60_lag3,
    LAG(i.ma5,  4) OVER w AS ma5_lag4,
    LAG(i.ma10, 4) OVER w AS ma10_lag4,
    LAG(i.ma20, 4) OVER w AS ma20_lag4,
    LAG(i.ma60, 4) OVER w AS ma60_lag4,
    LAG(i.ma5,  5) OVER w AS ma5_lag5,
    LAG(i.ma10, 5) OVER w AS ma10_lag5,
    LAG(i.ma20, 5) OVER w AS ma20_lag5,
    LAG(i.ma60, 5) OVER w AS ma60_lag5,
    LAG(i.dif,  1) OVER w AS dif_lag1,
    LAG(i.dea,  1) OVER w AS dea_lag1,
    LAG(i.k,    1) OVER w AS k_lag1,
    LAG(i.d,    1) OVER w AS d_lag1
  FROM stock_history_mv h
  LEFT JOIN stock_basic_info b ON b.symbol = h.symbol
  LEFT JOIN stock_indicators  i ON i.symbol = h.symbol AND i.calc_date = h.trade_date
  WHERE h.trade_date >= (SELECT MAX(trade_date) FROM stock_history_mv) - INTERVAL '90 days'
  WINDOW w AS (PARTITION BY h.symbol ORDER BY h.trade_date)
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
