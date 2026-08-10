// Package presets 把 JSONB 形态的选股规则表达式翻译成 PostgreSQL WHERE 子句，
// 并提供内置预设。
//
// 表达式 schema：
//
//	{
//	  "all":     [<condition>, ...],   // 全部满足
//	  "exclude": [<condition>, ...]    // 全部不满足
//	}
//
// 14 类 condition（见 compileOne）。详见 presets.go 中预设的写法。
package presets

import (
	"encoding/json"
	"fmt"
	"strings"
)

// expression 顶层 JSON。
type expression struct {
	All     []map[string]interface{} `json:"all"`
	Exclude []map[string]interface{} `json:"exclude"`
}

// CompileResult 生成的 SQL 片段与参数。
type CompileResult struct {
	Where string        // WHERE 子句（不含 WHERE 关键字），保证非空
	Args  []interface{} // 占位符参数
}

// Compile 把 JSONB 表达式编译成 WHERE 子句。
func Compile(exprJSON []byte) (CompileResult, error) {
	var e expression
	if len(exprJSON) == 0 {
		return CompileResult{}, fmt.Errorf("empty expression")
	}
	if err := json.Unmarshal(exprJSON, &e); err != nil {
		return CompileResult{}, fmt.Errorf("invalid json: %w", err)
	}
	if len(e.All) == 0 && len(e.Exclude) == 0 {
		return CompileResult{}, fmt.Errorf("expression must contain all or exclude")
	}

	parts := []string{"1=1"}
	args := []interface{}{}
	idx := 1
	for _, c := range e.All {
		sql, newArgs, used, err := compileOne(c, idx)
		if err != nil {
			return CompileResult{}, fmt.Errorf("all: %w", err)
		}
		if sql == "" {
			continue
		}
		parts = append(parts, sql)
		args = append(args, newArgs...)
		idx += used
	}
	for _, c := range e.Exclude {
		sql, newArgs, used, err := compileOne(c, idx)
		if err != nil {
			return CompileResult{}, fmt.Errorf("exclude: %w", err)
		}
		if sql == "" {
			continue
		}
		parts = append(parts, "NOT ("+sql+")")
		args = append(args, newArgs...)
		idx += used
	}
	where := strings.ReplaceAll(strings.Join(parts, " AND "), "ranked.", "latest.")
	return CompileResult{Where: where, Args: args}, nil
}

// compileOne 返回 (sql, args, placeholder_count, error)
// placeholder_count = 0 表示该谓词没有占位符（用了 LAG / 列直接比较）。
func compileOne(c map[string]interface{}, idx int) (string, []interface{}, int, error) {
	t, _ := c["type"].(string)
	switch t {

	// --- 通用字段比较 ---
	case "field":
		name, _ := c["name"].(string)
		op, _ := c["op"].(string)
		v, ok := numericArg(c["value"])
		if !ok {
			return "", nil, 0, fmt.Errorf("field: missing value")
		}
		col, err := resolveField(name)
		if err != nil {
			return "", nil, 0, err
		}
		opSQL, err := compareOp(op)
		if err != nil {
			return "", nil, 0, err
		}
		return fmt.Sprintf("latest.%s %s $%d", col, opSQL, idx), []interface{}{v}, 1, nil

	case "field_between":
		name, _ := c["name"].(string)
		minV, ok1 := numericArg(c["min"])
		maxV, ok2 := numericArg(c["max"])
		if !ok1 || !ok2 {
			return "", nil, 0, fmt.Errorf("field_between: need min & max")
		}
		col, err := resolveField(name)
		if err != nil {
			return "", nil, 0, err
		}
		return fmt.Sprintf("latest.%s BETWEEN $%d AND $%d", col, idx, idx+1),
			[]interface{}{minV, maxV}, 2, nil

	// --- 均线关系 ---
	case "ma_compare":
		fast, _ := c["fast"].(string)
		slow, _ := c["slow"].(string)
		op, _ := c["op"].(string)
		fastCol, err := resolveField(fast)
		if err != nil {
			return "", nil, 0, err
		}
		slowCol, err := resolveField(slow)
		if err != nil {
			return "", nil, 0, err
		}
		opSQL, err := compareOp(op)
		if err != nil {
			return "", nil, 0, err
		}
		return fmt.Sprintf("(latest.%s IS NOT NULL AND latest.%s IS NOT NULL AND latest.%s %s latest.%s)",
			fastCol, slowCol, fastCol, opSQL, slowCol), nil, 0, nil

	case "close_vs_ma":
		// 收盘价 vs 某条均线：ma=ma5/ma10/ma20/ma60，op=gt/gte/lt/lte
		ma, _ := c["ma"].(string)
		op, _ := c["op"].(string)
		maCol, err := resolveField(ma)
		if err != nil {
			return "", nil, 0, err
		}
		opSQL, err := compareOp(op)
		if err != nil {
			return "", nil, 0, err
		}
		return fmt.Sprintf("(latest.%s IS NOT NULL AND latest.close IS NOT NULL AND latest.close %s latest.%s)",
			maCol, opSQL, maCol), nil, 0, nil

	case "ma_alignment":
		// order: ["ma5","ma10","ma20","ma60"] 要求严格升序
		orderRaw, _ := c["order"].([]interface{})
		if len(orderRaw) < 2 {
			return "", nil, 0, fmt.Errorf("ma_alignment: need order[]")
		}
		cols := make([]string, 0, len(orderRaw))
		for _, x := range orderRaw {
			s, _ := x.(string)
			col, err := resolveField(s)
			if err != nil {
				return "", nil, 0, err
			}
			cols = append(cols, col)
		}
		var conds []string
		for i := 0; i < len(cols)-1; i++ {
			conds = append(conds, fmt.Sprintf("latest.%s > latest.%s", cols[i], cols[i+1]))
		}
		// 所有列都必须非 NULL
		nonNull := make([]string, len(cols))
		for i, c := range cols {
			nonNull[i] = fmt.Sprintf("latest.%s IS NOT NULL", c)
		}
		return "(" + strings.Join(nonNull, " AND ") + " AND " + strings.Join(conds, " AND ") + ")", nil, 0, nil

	case "ma_slope":
		// ma: ma5/ma10/... days: 5 op: gt
		// 表示 latest.ma > ranked.ma_N_days_ago
		ma, _ := c["ma"].(string)
		days, _ := numericArg(c["days"])
		op, _ := c["op"].(string)
		col, err := resolveField(ma)
		if err != nil {
			return "", nil, 0, err
		}
		opSQL, err := compareOp(op)
		if err != nil {
			return "", nil, 0, err
		}
		lagN := int(days)
		if lagN < 1 {
			lagN = 1
		}
		return fmt.Sprintf("ranked.%s_lag%d IS NOT NULL AND latest.%s IS NOT NULL AND latest.%s %s ranked.%s_lag%d",
			col, lagN, col, col, opSQL, col, lagN), nil, 0, nil

	// --- 量能 ---
	case "volume_ratio":
		// latest.volume / ranked.vol_avg5 >= min
		minV, ok := numericArg(c["min"])
		if !ok {
			return "", nil, 0, fmt.Errorf("volume_ratio: need min")
		}
		return fmt.Sprintf("ranked.vol_avg5 > 0 AND latest.volume / ranked.vol_avg5 >= $%d", idx),
			[]interface{}{minV}, 1, nil

	case "volume_increasing":
		// days: 3 min_ratio: 1.2  → V(t) > V(t-1) > V(t-2) AND V(t)/V(t-2) >= min_ratio
		days, _ := numericArg(c["days"])
		minRatio, ok := numericArg(c["min_ratio"])
		if !ok {
			minRatio = 1.0
		}
		d := int(days)
		if d < 2 {
			d = 2
		}
		conds := []string{
			fmt.Sprintf("ranked.vol_lag1 IS NOT NULL AND ranked.vol_lag%d IS NOT NULL", d),
			fmt.Sprintf("latest.volume > ranked.vol_lag1"),
		}
		for i := 2; i <= d; i++ {
			conds = append(conds, fmt.Sprintf("ranked.vol_lag%d > ranked.vol_lag%d", i-1, i))
		}
		conds = append(conds, fmt.Sprintf("latest.volume / ranked.vol_lag%d >= $%d", d, idx))
		return strings.Join(conds, " AND "), []interface{}{minRatio}, 1, nil

	// --- 窗口聚合 ---
	case "window_field":
		name, _ := c["name"].(string)
		days, _ := numericArg(c["days"])
		op, _ := c["op"].(string)
		col, err := resolveField(name)
		if err != nil {
			return "", nil, 0, err
		}
		// runner.go 的 ranked CTE 给部分字段用了短别名（net_amount → net、volume → vol），
		// 这里生成 SQL 必须使用同一前缀，否则会报 "column does not exist"。
		lagPrefix := lagAlias(col)
		d := int(days)
		if d < 1 {
			d = 1
		}
		// 注：CTE 中 lag 索引从 1 开始（LAG(x, 1) = 前一天），lag0 不存在。
		// 这里把 "days=d" 翻译成 ranked.<prefix>_lag1..lag{d}，共 d 条比较。
		switch op {
		case "always_positive":
			var conds []string
			for i := 1; i <= d; i++ {
				conds = append(conds, fmt.Sprintf("ranked.%s_lag%d > 0", lagPrefix, i))
			}
			return strings.Join(conds, " AND "), nil, 0, nil
		case "always_negative":
			var conds []string
			for i := 1; i <= d; i++ {
				conds = append(conds, fmt.Sprintf("ranked.%s_lag%d < 0", lagPrefix, i))
			}
			return strings.Join(conds, " AND "), nil, 0, nil
		default:
			return "", nil, 0, fmt.Errorf("window_field: unknown op %q", op)
		}

	case "yang_streak":
		days, _ := numericArg(c["days"])
		d := int(days)
		if d < 1 {
			d = 1
		}
		var conds []string
		for i := 0; i < d; i++ {
			conds = append(conds, fmt.Sprintf("ranked.yang_lag%d = TRUE", i))
		}
		return strings.Join(conds, " AND "), nil, 0, nil

	case "cumulative_change":
		// days: 3 max_pct: 15  → (close(t) - close(t-N)) / close(t-N) * 100 <= max_pct
		days, _ := numericArg(c["days"])
		maxPct, ok := numericArg(c["max_pct"])
		if !ok {
			return "", nil, 0, fmt.Errorf("cumulative_change: need max_pct")
		}
		d := int(days)
		if d < 1 {
			d = 1
		}
		return fmt.Sprintf("ranked.close_lag%d IS NOT NULL AND ranked.close_lag%d > 0 AND ((latest.close - ranked.close_lag%d) / ranked.close_lag%d * 100) <= $%d",
			d, d, d, d, idx),
			[]interface{}{maxPct}, 1, nil

	// --- 突破 / 交叉 ---
	case "breakout_high":
		// close(t) > max(high(t-1)..high(t-N))
		days, _ := numericArg(c["lookback"])
		d := int(days)
		if d < 1 {
			d = 1
		}
		return fmt.Sprintf("ranked.high_max%d IS NOT NULL AND latest.close > ranked.high_max%d", d, d), nil, 0, nil

	case "macd_cross":
		// DIF crosses above DEA, location: below_zero | above_zero | any
		loc, _ := c["location"].(string)
		cond := "ranked.dif_lag1 IS NOT NULL AND ranked.dea_lag1 IS NOT NULL AND ranked.dif_lag1 <= ranked.dea_lag1 AND latest.dif > latest.dea"
		switch loc {
		case "below_zero":
			cond += " AND latest.dif < 0 AND latest.dea < 0"
		case "above_zero":
			cond += " AND latest.dif > 0 AND latest.dea > 0"
		}
		return cond, nil, 0, nil

	case "kdj_cross":
		// K crosses above D
		loc, _ := c["location"].(string)
		cond := "ranked.k_lag1 IS NOT NULL AND ranked.d_lag1 IS NOT NULL AND ranked.k_lag1 <= ranked.d_lag1 AND latest.k > latest.d"
		switch loc {
		case "below_20":
			cond += " AND latest.k < 20 AND latest.d < 20"
		case "above_80":
			cond += " AND latest.k > 80 AND latest.d > 80"
		}
		return cond, nil, 0, nil

	case "rsi_range":
		field, _ := c["field"].(string)
		minV, ok1 := numericArg(c["min"])
		maxV, ok2 := numericArg(c["max"])
		if !ok1 || !ok2 {
			return "", nil, 0, fmt.Errorf("rsi_range: need min/max")
		}
		col, err := resolveField(field)
		if err != nil {
			return "", nil, 0, err
		}
		return fmt.Sprintf("latest.%s BETWEEN $%d AND $%d", col, idx, idx+1),
			[]interface{}{minV, maxV}, 2, nil

	case "boll_position":
		// lower | upper | middle
		pos, _ := c["position"].(string)
		switch pos {
		case "lower":
			return "latest.boll_lower IS NOT NULL AND latest.close <= latest.boll_lower", nil, 0, nil
		case "upper":
			return "latest.boll_upper IS NOT NULL AND latest.close >= latest.boll_upper", nil, 0, nil
		case "middle":
			return "latest.boll_mid IS NOT NULL AND latest.close BETWEEN latest.boll_lower AND latest.boll_upper", nil, 0, nil
		}
		return "", nil, 0, fmt.Errorf("boll_position: bad position %q", pos)

	// --- 标的特征 ---
	case "is_st":
		return "(latest.name LIKE '%ST%' OR latest.name LIKE '%st%')", nil, 0, nil

	case "is_not_st":
		return "latest.name NOT LIKE '%ST%' AND latest.name NOT LIKE '%st%'", nil, 0, nil

	case "list_age_days_gte":
		// listing_date 距今 >= days 天
		// 注意：PG 在 prepare 阶段无法推断 $1 的类型，会把 "CURRENT_DATE - $1" 解析成
		// "timestamptz - int"，报错 "operator does not exist: timestamp with time zone > integer"。
		// 显式 cast 成 int 后 -> "date - int = date"，可继续与 listing_date 比较。
		days, _ := numericArg(c["days"])
		d := int(days)
		return fmt.Sprintf("basic.listing_date IS NOT NULL AND basic.listing_date <= (CURRENT_DATE - $%d::int)", idx),
			[]interface{}{d}, 1, nil

	case "list_age_days_lt":
		days, _ := numericArg(c["days"])
		d := int(days)
		// 排除「上市未满 N 天」：只有真正有 listing_date 且距今 < N 天的股票才被标记为新股。
		// listing_date 为 NULL 表示未知上市日期，按老股放行。
		// 作为 exclude 时（默认）：整体 NOT 后只有「真新股」被排除，老股与未知都保留。
		return fmt.Sprintf("(basic.listing_date IS NOT NULL AND basic.listing_date > (CURRENT_DATE - $%d::int))", idx),
			[]interface{}{d}, 1, nil

	case "market_cap_yi":
		// 流通市值（亿元） = latest.close * basic.outstanding_shares / 1e8
		minV, ok1 := numericArg(c["min"])
		maxV, ok2 := numericArg(c["max"])
		if !ok1 || !ok2 {
			return "", nil, 0, fmt.Errorf("market_cap_yi: need min/max")
		}
		return fmt.Sprintf("(basic.outstanding_shares IS NOT NULL AND basic.outstanding_shares > 0 AND latest.close * basic.outstanding_shares / 1e8 BETWEEN $%d AND $%d)", idx, idx+1),
			[]interface{}{minV, maxV}, 2, nil

	// --- 板型筛选 ---
	case "board_in":
		raw, _ := c["boards"].([]interface{})
		if len(raw) == 0 {
			return "", nil, 0, fmt.Errorf("board_in: need boards[]")
		}
		conds := []string{}
		for _, b := range raw {
			s, _ := b.(string)
			switch s {
			case "科创板":
				conds = append(conds, "latest.symbol LIKE '688%'")
			case "主板":
				conds = append(conds, "(latest.symbol LIKE '60%' OR latest.symbol LIKE '00%' OR latest.symbol LIKE '20%')")
			case "创业板":
				conds = append(conds, "(latest.symbol LIKE '300%' OR latest.symbol LIKE '301%')")
			case "B股":
				conds = append(conds, "latest.symbol LIKE '9%'")
			case "北交所":
				conds = append(conds, "(latest.symbol LIKE '8%' OR latest.symbol LIKE '43%' OR latest.symbol LIKE '92%')")
			default:
				return "", nil, 0, fmt.Errorf("board_in: unknown board %q", s)
			}
		}
		return "(" + strings.Join(conds, " OR ") + ")", nil, 0, nil

	default:
		return "", nil, 0, fmt.Errorf("unknown condition type %q", t)
	}
}

// resolveField 把规则字段名映射成 ranked/latest 子查询里的列名。
func resolveField(name string) (string, error) {
	switch name {
	case "close", "open", "high", "low", "volume",
		"change_percent", "turnover_rate", "net_amount",
		"in_amount", "out_amount":
		return name, nil
	case "pe_ttm":
		return "pettm", nil
	case "pb":
		return "pb", nil
	case "ma5", "ma10", "ma20", "ma60":
		return name, nil
	case "macd", "dif", "dea":
		return name, nil
	case "rsi6", "rsi12", "rsi24":
		return name, nil
	case "k", "d", "j":
		return name, nil
	case "boll_upper", "boll_mid", "boll_lower":
		return name, nil
	}
	return "", fmt.Errorf("unknown field %q", name)
}

func compareOp(op string) (string, error) {
	switch op {
	case "gt":
		return ">", nil
	case "gte":
		return ">=", nil
	case "lt":
		return "<", nil
	case "lte":
		return "<=", nil
	case "eq":
		return "=", nil
	}
	return "", fmt.Errorf("bad op %q", op)
}

func numericArg(v interface{}) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case json.Number:
		f, _ := x.Float64()
		return f, true
	}
	return 0, false
}

// lagAlias 把 resolveField 输出映射成 ranked CTE 中的 lag 列前缀。
// 必须与 runner.go 中 LAG(... ) AS xxx_lagN 的别名保持一致。
func lagAlias(col string) string {
	switch col {
	case "net_amount":
		return "net"
	case "volume":
		return "vol"
	}
	return col
}
