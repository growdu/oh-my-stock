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
		Description: "当前模型：PE-TTM 0-200（多数股缺 PE，放宽到 200），覆盖主板/创业板/科创板。",
		Expression: map[string]interface{}{
			"all": []map[string]interface{}{
				boardFilter([]string{"主板", "创业板", "科创板"}),
				{"type": "field_between", "name": "pe_ttm", "min": 0, "max": 200},
			},
			"exclude": commonExcludes(),
		},
	},
	{
		ID: "star-board", Name: "科创板机会",
		Description: "仅看科创板（688xxx），量比 ≥1.2、站上 MA5。",
		Expression: map[string]interface{}{
			"all": []map[string]interface{}{
				boardFilter([]string{"科创板"}),
				{"type": "volume_ratio", "min": 1.2},
				{"type": "close_vs_ma", "ma": "ma5", "op": "gt"},
			},
			"exclude": commonExcludes(),
		},
	},
	{
		ID: "chinext", Name: "创业板机会",
		Description: "仅看创业板（300/301xxx），KDJ 金叉 + 量比 ≥1.2。",
		Expression: map[string]interface{}{
			"all": []map[string]interface{}{
				boardFilter([]string{"创业板"}),
				{"type": "kdj_cross", "location": "any"},
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
