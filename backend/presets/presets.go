package presets

type Preset struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Expression  map[string]interface{} `json:"expression"`
}

func commonExcludes() []map[string]interface{} {
	return []map[string]interface{}{
		{"type": "is_st"},
		{"type": "list_age_days_lt", "days": 60},
	}
}

func boardFilter(bs []string) map[string]interface{} {
	return map[string]interface{}{"type": "board_in", "boards": bs}
}

var All = []Preset{
	{
		ID: "bottom-reversal", Name: "底部反转",
		Description: "近 2 日资金净流入、连阳，站上 MA5，KDJ 金叉。",
		Expression: map[string]interface{}{
			"all": []map[string]interface{}{
				boardFilter([]string{"主板", "创业板", "科创板"}),
				{"type": "window_field", "name": "net_amount", "op": "always_positive", "days": 2},
				{"type": "yang_streak", "days": 2},
				{"type": "close_vs_ma", "ma": "ma5", "op": "gt"},
				{"type": "kdj_cross", "location": "any"},
			},
			"exclude": commonExcludes(),
		},
	},
	{
		ID: "breakout-5d", Name: "突破近 5 日高点",
		Description: "收盘突破近 5 日最高且放量。",
		Expression: map[string]interface{}{
			"all": []map[string]interface{}{
				boardFilter([]string{"主板", "创业板", "科创板"}),
				{"type": "breakout_high", "lookback": 5},
				{"type": "volume_ratio", "min": 1.2},
			},
			"exclude": commonExcludes(),
		},
	},
	{
		ID: "ma-trend", Name: "MA5>MA10 多头",
		Description: "MA5 高于 MA10，且 MA5 较 2 日前抬高。",
		Expression: map[string]interface{}{
			"all": []map[string]interface{}{
				boardFilter([]string{"主板", "创业板", "科创板"}),
				{"type": "ma_compare", "fast": "ma5", "slow": "ma10", "op": "gt"},
				{"type": "ma_slope", "ma": "ma5", "days": 2, "op": "gt"},
			},
			"exclude": commonExcludes(),
		},
	},
	{
		ID: "volume-price", Name: "量价齐升",
		Description: "量比 ≥1.2，站上 MA5 且 MA5 向上。",
		Expression: map[string]interface{}{
			"all": []map[string]interface{}{
				boardFilter([]string{"主板", "创业板", "科创板"}),
				{"type": "volume_ratio", "min": 1.2},
				{"type": "close_vs_ma", "ma": "ma5", "op": "gt"},
				{"type": "ma_slope", "ma": "ma5", "days": 2, "op": "gt"},
			},
			"exclude": commonExcludes(),
		},
	},
	{
		ID: "tech-bounce", Name: "技术反弹",
		Description: "KDJ 金叉 + RSI6 在 30-60 + 站上 MA5。",
		Expression: map[string]interface{}{
			"all": []map[string]interface{}{
				boardFilter([]string{"主板", "创业板", "科创板"}),
				{"type": "kdj_cross", "location": "any"},
				{"type": "rsi_range", "field": "rsi6", "min": 30, "max": 60},
				{"type": "close_vs_ma", "ma": "ma5", "op": "gt"},
			},
			"exclude": commonExcludes(),
		},
	},
	{
		ID: "quality-stocks", Name: "稳健基本面",
		Description: "PE-TTM 0~80、PB<10 的主板/创业板/科创板，要求估值字段非空。",
		Expression: map[string]interface{}{
			"all": []map[string]interface{}{
				boardFilter([]string{"主板", "创业板", "科创板"}),
				{"type": "field_between", "name": "pe_ttm", "min": 0, "max": 80},
				{"type": "field", "name": "pb", "op": "lt", "value": 10},
			},
			"exclude": commonExcludes(),
		},
	},
	{
		ID: "ma-golden-cross", Name: "均线金叉",
		Description: "MA5 上穿 MA10，同时站上 MA20；属于典型多头启动信号。",
		Expression: map[string]interface{}{
			"all": []map[string]interface{}{
				boardFilter([]string{"主板", "创业板", "科创板"}),
				{"type": "ma_cross", "fast": "ma5", "slow": "ma10", "direction": "golden"},
				{"type": "ma_compare", "fast": "ma5", "slow": "ma20", "op": "gt"},
				{"type": "ma_compare", "fast": "ma10", "slow": "ma20", "op": "gt"},
				{"type": "ma_slope", "ma": "ma20", "days": 3, "op": "gt"},
			},
			"exclude": commonExcludes(),
		},
	},
	{
		ID: "ma-death-cross", Name: "均线死叉",
		Description: "MA5 下穿 MA20，进入空头排列；适合风险规避。",
		Expression: map[string]interface{}{
			"all": []map[string]interface{}{
				boardFilter([]string{"主板", "创业板", "科创板"}),
				{"type": "ma_cross", "fast": "ma5", "slow": "ma20", "direction": "death"},
				{"type": "ma_compare", "fast": "ma5", "slow": "ma60", "op": "lt"},
			},
			"exclude": commonExcludes(),
		},
	},
	{
		ID: "oversold-bounce", Name: "超卖反弹",
		Description: "RSI6<30 触底 + KDJ 金叉 + 站上 MA5，典型短线反弹。",
		Expression: map[string]interface{}{
			"all": []map[string]interface{}{
				boardFilter([]string{"主板", "创业板", "科创板"}),
				{"type": "field", "name": "rsi6", "op": "lt", "value": 30},
				{"type": "kdj_cross", "location": "any"},
				{"type": "close_vs_ma", "ma": "ma5", "op": "gt"},
			},
			"exclude": commonExcludes(),
		},
	},
	{
		ID: "ma-converge", Name: "均线粘合",
		Description: "MA5/10/20 三线粘合（最大与最小乖离<2%），等待方向选择。",
		Expression: map[string]interface{}{
			"all": []map[string]interface{}{
				boardFilter([]string{"主板", "创业板", "科创板"}),
				{"type": "ma_alignment", "order": []string{"ma5", "ma10", "ma20"}},
				{"type": "bias", "ma": "ma20", "min": -2, "max": 2},
				{"type": "turnover_rate_range", "min": 1, "max": 15},
			},
			"exclude": commonExcludes(),
		},
	},
	{
		ID: "boll-bounce", Name: "BOLL 中轨反弹",
		Description: "BOLL 下轨附近 + 站上中轨 + MACD 柱转正，趋势拐点。",
		Expression: map[string]interface{}{
			"all": []map[string]interface{}{
				boardFilter([]string{"主板", "创业板", "科创板"}),
				{"type": "boll_position", "position": "lower"},
				{"type": "close_vs_ma", "ma": "ma5", "op": "gt"},
				{"type": "macd_cross", "location": "above_zero"},
			},
			"exclude": commonExcludes(),
		},
	},
	{
		ID: "volume-shrink-pullback", Name: "缩量回踩 MA20",
		Description: "MA5 在 MA20 之上、收盘回踩 MA5 之下、量比<=0.8、换手<5%；强势洗盘形态。",
		Expression: map[string]interface{}{
			"all": []map[string]interface{}{
				boardFilter([]string{"主板", "创业板", "科创板"}),
				{"type": "ma_compare", "fast": "ma5", "slow": "ma20", "op": "gt"},
				{"type": "close_vs_ma", "ma": "ma20", "op": "gt"},
				{"type": "close_vs_ma", "ma": "ma5", "op": "lt"},
				{"type": "volume_ratio", "max": 0.8},
				{"type": "turnover_rate_range", "min": 0, "max": 5},
			},
			"exclude": commonExcludes(),
		},
	},
	{
		ID: "limit-up-strong", Name: "强势涨停",
		Description: "今日涨停（>=9.8%）+ 站上 MA20 + 量比>=1.5。",
		Expression: map[string]interface{}{
			"all": []map[string]interface{}{
				boardFilter([]string{"主板", "创业板", "科创板"}),
				{"type": "limit_up", "min_pct": 9.8},
				{"type": "close_vs_ma", "ma": "ma20", "op": "gt"},
				{"type": "volume_ratio", "min": 1.5},
			},
			"exclude": commonExcludes(),
		},
	},
	{
		ID: "high-position-breakout", Name: "高位突破",
		Description: "收盘在 60 日区间 [80%, 100%] + MA 多头排列 + 量比放大。",
		Expression: map[string]interface{}{
			"all": []map[string]interface{}{
				boardFilter([]string{"主板", "创业板", "科创板"}),
				{"type": "close_position", "lookback": 60, "min": 0.8, "max": 1.0},
				{"type": "ma_alignment", "order": []string{"ma5", "ma10", "ma20", "ma60"}},
				{"type": "volume_ratio", "min": 1.2},
			},
			"exclude": commonExcludes(),
		},
	},
}

func ByID(id string) *Preset {
	for i := range All {
		if All[i].ID == id {
			return &All[i]
		}
	}
	return nil
}
