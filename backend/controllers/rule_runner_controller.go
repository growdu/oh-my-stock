package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"oh-my-stock/config"
	"oh-my-stock/middleware"
	"oh-my-stock/models"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ============================================================
// 规则执行器：JSONB → stock_history_mv → 写入 target_trend_stock
//
// 规则表达式例子：
// {
//   "market": "创业板",                          // 直接等于
//   "industry": {"in": ["银行","证券"]},         // IN
//   "industry": {"eq": "银行"},                  // =
//   "symbol_prefix": "300",                      // LIKE '300%'
//   "change_percent": {"gt": 5, "lt": 9.8},      // 区间
//   "turnover_rate": {"gte": 3},
//   "current_price": {"between": [5, 50]},
//   "consecutive_up_days":   {"gte": 3},          // 连续 N 天上涨
//   "consecutive_inflow_days":{"gte": 3},          // 连续 N 天主力净流入
//   "consecutive_volume_amplify_days": {"gte": 3}, // 连续 N 天放量
//   "volume_amplify_days":   {"gte": 3, "min_ratio": 1.5} // 连续放量倍数
// }
// ============================================================

// RunRule 执行已存在的 user_stock_rule
func RunRule(c *gin.Context) {
	uid := middleware.GetUserID(c)
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	var rule models.UserStockRule
	if err := config.DB.Where("id = ? AND user_id = ?", c.Param("id"), uid).First(&rule).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "规则不存在"})
		return
	}
	matched := runRuleCore(rule)
	c.JSON(http.StatusOK, gin.H{
		"matched": len(matched),
		"date":    time.Now().Format("2006-01-02"),
		"rules":   matched,
	})
}

// PreviewRule 预览（不入库）
func PreviewRule(c *gin.Context) {
	var req struct {
		RuleName       string                 `json:"rule_name" binding:"required"`
		RuleExpression map[string]interface{} `json:"rule_expression" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	b, _ := json.Marshal(req.RuleExpression)
	tmp := models.UserStockRule{
		RuleName:       req.RuleName,
		RuleExpression: b,
		UserID:         middleware.GetUserID(c),
	}
	matched := runRuleCore(tmp)
	c.JSON(http.StatusOK, gin.H{"matched": len(matched), "rules": matched})
}

// ListTargetStocks 查询候选股
func ListTargetStocks(c *gin.Context) {
	var rows []models.TargetTrendStock
	q := config.DB.Model(&models.TargetTrendStock{}).
		Where("matched_at = ?", time.Now().Format("2006-01-02"))
	if ruleName := c.Query("rule_name"); ruleName != "" {
		q = q.Where("rule_name = ?", ruleName)
	}
	q.Order("change_percent DESC").Limit(200).Find(&rows)
	c.JSON(http.StatusOK, gin.H{"total": len(rows), "data": rows})
}

// ----------------------------------------------------------
// 核心：解析 + 查询 + 入库
// ----------------------------------------------------------
func runRuleCore(rule models.UserStockRule) []models.TargetTrendStock {
	expr := map[string]interface{}{}
	_ = json.Unmarshal(rule.RuleExpression, &expr)

	amplifyRatio := getFloat(expr["volume_amplify_days"], "min_ratio", 1.2)
	whereSQL, args := buildWhereFromSpec(expr, "h", amplifyRatio)

	baseSQL := fmt.Sprintf(`
		WITH latest AS (
		    SELECT symbol, MAX(trade_date) AS trade_date
		    FROM stock_history_mv
		    WHERE trade_date >= CURRENT_DATE - INTERVAL '60 days'
		    GROUP BY symbol
		),
		streak_up AS (
		    SELECT symbol, COUNT(*) AS days
		    FROM (
		        SELECT symbol, trade_date,
		               ROW_NUMBER() OVER (PARTITION BY symbol ORDER BY trade_date DESC) AS rn,
		               SUM(CASE WHEN change_percent > 0 THEN 1 ELSE 0 END)
		                   OVER (PARTITION BY symbol ORDER BY trade_date DESC
		                         ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS cum
		        FROM stock_history_mv
		        WHERE trade_date >= CURRENT_DATE - INTERVAL '60 days'
		    ) t
		    WHERE cum = rn
		    GROUP BY symbol, cum
		),
		streak_in AS (
		    SELECT symbol, COUNT(*) AS days
		    FROM (
		        SELECT symbol, trade_date,
		               ROW_NUMBER() OVER (PARTITION BY symbol ORDER BY trade_date DESC) AS rn,
		               SUM(CASE WHEN net_amount > 0 THEN 1 ELSE 0 END)
		                   OVER (PARTITION BY symbol ORDER BY trade_date DESC
		                         ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS cum
		        FROM stock_history_mv
		        WHERE trade_date >= CURRENT_DATE - INTERVAL '60 days'
		    ) t
		    WHERE cum = rn
		    GROUP BY symbol, cum
		),
		streak_vol AS (
		    SELECT symbol, COUNT(*) AS days
		    FROM (
		        SELECT symbol, trade_date, prev_vol, rn,
		               SUM(CASE WHEN (volume > 0 AND prev_vol IS NOT NULL
		                              AND volume >= prev_vol * $1) THEN 1 ELSE 0 END)
		                   OVER (PARTITION BY symbol ORDER BY trade_date DESC
		                         ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS cum
		        FROM (
		            SELECT symbol, trade_date, volume,
		                   LAG(volume) OVER (PARTITION BY symbol ORDER BY trade_date DESC) AS prev_vol,
		                   ROW_NUMBER() OVER (PARTITION BY symbol ORDER BY trade_date DESC) AS rn
		            FROM stock_history_mv
		            WHERE trade_date >= CURRENT_DATE - INTERVAL '60 days'
		        ) inner_t
		    ) t
		    WHERE cum = rn
		    GROUP BY symbol, cum
		)
		SELECT h.symbol, h.name, h.industry, h.market,
		       h.close AS current_price,
		       h.change_percent,
		       h.turnover_rate,
		       h.net_amount,
		       COALESCE(su.days, 0) AS consecutive_up_days,
		       COALESCE(si.days, 0) AS consecutive_inflow_days,
		       COALESCE(sv.days, 0) AS consecutive_volume_amplify_days,
		       COALESCE(cur.ma5, 0)        AS ma5,
		       COALESCE(cur.ma10, 0)       AS ma10,
		       COALESCE(cur.ma20, 0)       AS ma20,
		       COALESCE(cur.ma60, 0)       AS ma60,
		       COALESCE(cur.macd, 0)       AS macd,
		       COALESCE(cur.dif, 0)        AS dif,
		       COALESCE(cur.dea, 0)        AS dea,
		       COALESCE(cur.k, 0)          AS k,
		       COALESCE(cur.d, 0)          AS d,
		       COALESCE(cur.j, 0)          AS j,
		       COALESCE(cur.rsi6, 0)       AS rsi6,
		       COALESCE(cur.rsi12, 0)      AS rsi12,
		       COALESCE(cur.rsi24, 0)      AS rsi24,
		       COALESCE(cur.boll_upper, 0) AS boll_upper,
		       COALESCE(cur.boll_mid, 0)   AS boll_mid,
		       COALESCE(cur.boll_lower, 0) AS boll_lower,
		       COALESCE(prev.ma5, 0)       AS ma5_prev,
		       COALESCE(prev.ma10, 0)      AS ma10_prev,
		       COALESCE(prev.ma20, 0)      AS ma20_prev
		FROM stock_history_mv h
		JOIN latest l ON l.symbol = h.symbol AND l.trade_date = h.trade_date
		LEFT JOIN streak_up   su ON su.symbol = h.symbol
		LEFT JOIN streak_in   si ON si.symbol = h.symbol
		LEFT JOIN streak_vol  sv ON sv.symbol = h.symbol
		LEFT JOIN LATERAL (
		    SELECT ma5, ma10, ma20, ma60, macd, dif, dea, k, d, j,
		           rsi6, rsi12, rsi24, boll_upper, boll_mid, boll_lower
		    FROM stock_indicators
		    WHERE symbol = h.symbol AND calc_date = h.trade_date
		    LIMIT 1
		) cur ON TRUE
		LEFT JOIN LATERAL (
		    SELECT ma5, ma10, ma20
		    FROM stock_indicators
		    WHERE symbol = h.symbol AND calc_date < h.trade_date
		    ORDER BY calc_date DESC
		    LIMIT 1
		) prev ON TRUE
		WHERE 1=1 %s
		ORDER BY h.change_percent DESC
		LIMIT 200
	`, whereSQL)

	type rawRow struct {
		Symbol                       string  `gorm:"column:symbol"`
		Name                         string  `gorm:"column:name"`
		Industry                     string  `gorm:"column:industry"`
		Market                       string  `gorm:"column:market"`
		CurrentPrice                 float64 `gorm:"column:current_price"`
		ChangePercent                float64 `gorm:"column:change_percent"`
		TurnoverRate                 float64 `gorm:"column:turnover_rate"`
		NetAmount                    float64 `gorm:"column:net_amount"`
		ConsecutiveUpDays            int     `gorm:"column:consecutive_up_days"`
		ConsecutiveInflowDays        int     `gorm:"column:consecutive_inflow_days"`
		ConsecutiveVolumeAmplifyDays int     `gorm:"column:consecutive_volume_amplify_days"`
		MA5                          float64 `gorm:"column:ma5"`
		MA10                         float64 `gorm:"column:ma10"`
		MA20                         float64 `gorm:"column:ma20"`
		MA60                         float64 `gorm:"column:ma60"`
		MA5Prev                      float64 `gorm:"column:ma5_prev"`
		MA10Prev                     float64 `gorm:"column:ma10_prev"`
		MA20Prev                     float64 `gorm:"column:ma20_prev"`
		MACD                         float64 `gorm:"column:macd"`
		DIF                          float64 `gorm:"column:dif"`
		DEA                          float64 `gorm:"column:dea"`
		K                            float64 `gorm:"column:k"`
		D                            float64 `gorm:"column:d"`
		J                            float64 `gorm:"column:j"`
		RSI6                         float64 `gorm:"column:rsi6"`
		RSI12                        float64 `gorm:"column:rsi12"`
		RSI24                        float64 `gorm:"column:rsi24"`
		BollUpper                    float64 `gorm:"column:boll_upper"`
		BollMid                      float64 `gorm:"column:boll_mid"`
		BollLower                    float64 `gorm:"column:boll_lower"`
	}
	var rows []rawRow
	if err := config.DB.Raw(baseSQL, args...).Scan(&rows).Error; err != nil {
		return nil
	}

	matched := make([]models.TargetTrendStock, 0, len(rows))
	today := time.Now().Truncate(24 * time.Hour)
	for _, r := range rows {
		rid := rule.ID
		matched = append(matched, models.TargetTrendStock{
			Symbol:        r.Symbol,
			Name:          r.Name,
			RuleName:      rule.RuleName,
			RuleID:        &rid,
			UserID:        rule.UserID,
			CurrentPrice:  r.CurrentPrice,
			ChangePercent: r.ChangePercent,
			TurnoverRate:  r.TurnoverRate,
			NetInflow:     r.NetAmount,
			Industry:      r.Industry,
			Market:        r.Market,
			MatchedAt:     today,
			MA5:           r.MA5,
			MA10:          r.MA10,
			MA20:          r.MA20,
			MA60:          r.MA60,
			MA5Prev:       r.MA5Prev,
			MA10Prev:      r.MA10Prev,
			MA20Prev:      r.MA20Prev,
			MACD:          r.MACD,
			DIF:           r.DIF,
			DEA:           r.DEA,
			K:             r.K,
			D:             r.D,
			J:             r.J,
			RSI6:          r.RSI6,
			RSI12:         r.RSI12,
			RSI24:         r.RSI24,
			BollUpper:     r.BollUpper,
			BollMid:       r.BollMid,
			BollLower:     r.BollLower,
		})
	}
	if len(matched) > 0 {
		config.DB.Where("rule_name = ? AND matched_at = ?", rule.RuleName, today).
			Delete(&models.TargetTrendStock{})
		for i := range matched {
			config.DB.Create(&matched[i])
		}
	}
	return matched
}

// ----------------------------------------------------------
// 工具
// ----------------------------------------------------------
type cmpItem struct{ col, op string; v interface{} }

func buildWhereFromSpec(spec map[string]interface{}, alias string, amplifyRatio float64) (string, []interface{}) {
	args := []interface{}{amplifyRatio}; _ = args // $1 = amplifyRatio (CTE 用); outer placeholder starts at $2
	ph := func() string { return fmt.Sprintf("$%d", len(args)+1) }
	var conds []string

	addCmp := func(col string, v interface{}) {
		for _, it := range cmpSlots(col, v) {
			conds = append(conds, fmt.Sprintf("%s %s %s", it.col, it.op, ph()))
			args = append(args, it.v)
		}
	}

	for k, v := range spec {
		switch k {
		case "volume_amplify_days":
			continue
		case "consecutive_up_days":
			addCmp("su.days", v)
		case "consecutive_inflow_days":
			addCmp("si.days", v)
		case "consecutive_volume_amplify_days":
			addCmp("sv.days", v)
		case "symbol_prefix":
			if s, ok := v.(string); ok {
				conds = append(conds, fmt.Sprintf("%s.symbol LIKE %s", alias, ph()))
				args = append(args, s+"%")
			}
		case "industry":
			if m, ok := v.(map[string]interface{}); ok {
				if arr, ok := m["in"].([]interface{}); ok && len(arr) > 0 {
					phs := make([]string, 0, len(arr))
					for _, x := range arr {
						phs = append(phs, ph())
						args = append(args, x)
					}
					conds = append(conds, fmt.Sprintf("%s.industry IN (%s)", alias, strings.Join(phs, ",")))
				} else if s, ok := m["eq"].(string); ok {
					conds = append(conds, fmt.Sprintf("%s.industry = %s", alias, ph()))
					args = append(args, s)
				}
			} else if s, ok := v.(string); ok {
				conds = append(conds, fmt.Sprintf("%s.industry = %s", alias, ph()))
				args = append(args, s)
			}
		case "market":
			if m, ok := v.(map[string]interface{}); ok {
				if arr, ok := m["in"].([]interface{}); ok && len(arr) > 0 {
					phs := make([]string, 0, len(arr))
					for _, x := range arr {
						phs = append(phs, ph())
						args = append(args, x)
					}
					conds = append(conds, fmt.Sprintf("%s.market IN (%s)", alias, strings.Join(phs, ",")))
				} else if s, ok := m["eq"].(string); ok {
					conds = append(conds, fmt.Sprintf("%s.market = %s", alias, ph()))
					args = append(args, s)
				}
			} else if s, ok := v.(string); ok {
				conds = append(conds, fmt.Sprintf("%s.market = %s", alias, ph()))
				args = append(args, s)
			}
		default:
			addCmp(alias+"."+k, v)
		}
	}
	if len(conds) == 0 {
		return "", args
	}
	return " AND " + strings.Join(conds, " AND "), args
}

func cmpSlots(col string, v interface{}) []cmpItem {
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	var items []cmpItem
	if x, ok := numFrom(m["gt"]); ok {
		items = append(items, cmpItem{col, ">", x})
	}
	if x, ok := numFrom(m["gte"]); ok {
		items = append(items, cmpItem{col, ">=", x})
	}
	if x, ok := numFrom(m["lt"]); ok {
		items = append(items, cmpItem{col, "<", x})
	}
	if x, ok := numFrom(m["lte"]); ok {
		items = append(items, cmpItem{col, "<=", x})
	}
	if x, ok := numFrom(m["eq"]); ok {
		items = append(items, cmpItem{col, "=", x})
	}
	if arr, ok := m["between"].([]interface{}); ok && len(arr) == 2 {
		a, _ := numFrom(arr[0])
		b, _ := numFrom(arr[1])
		items = append(items, cmpItem{col, ">=", a})
		items = append(items, cmpItem{col, "<=", b})
	}
	return items
}

func numFrom(v interface{}) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	}
	return 0, false
}

func getFloat(m interface{}, key string, def float64) float64 {
	if mp, ok := m.(map[string]interface{}); ok {
		if v, ok := numFrom(mp[key]); ok {
			return v
		}
	}
	return def
}
