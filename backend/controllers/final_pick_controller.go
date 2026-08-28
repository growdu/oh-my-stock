package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"oh-my-stock/config"
	"oh-my-stock/models"
	"oh-my-stock/presets"
)

// ---------- 请求 / 响应 ----------

type FinalPickRequest struct {
	PresetIDs []string `json:"preset_ids" binding:"required"`
	TopN      int      `json:"top_n"`
	TradeDate string   `json:"trade_date"` // 可选；缺省 = DB 最新
}

type ScoreBreakdown struct {
	Board       int `json:"board"`        // 板型加权（创业板 +5 / 科创板 +5 / 主板 +2）
	Technical   int `json:"technical"`    // 技术形态 (max 30)
	Momentum    int `json:"momentum"`     // 动量位置 (max 20)
	VolumePrice int `json:"volume_price"` // 量价健康 (max 18)
	Growth      int `json:"growth"`       // 成长性 (max 12)
	Fund        int `json:"fund"`         // 资金面 (max 10)
	Penalty     int `json:"penalty"`      // 减分合计（负数）
}

type PickStock struct {
	Symbol        string         `json:"symbol"`
	Name          string         `json:"name"`
	Industry      string         `json:"industry"`
	Market        string         `json:"market"`
	TradeDate     string         `json:"trade_date"`
	Score         int            `json:"score"`
	Breakdown     ScoreBreakdown `json:"breakdown"`
	MatchedPresets []string      `json:"matched_presets"`

	// 评分用到的关键字段（前端展示明细）
	Close         float64  `json:"close"`
	ChangePercent float64  `json:"change_percent"`
	TurnoverRate  float64  `json:"turnover_rate"`
	PETTM         float64  `json:"pe_ttm"`
	PB            float64  `json:"pb"`
	NetAmount     float64  `json:"net_amount"`
	MfChangePct   float64  `json:"mf_change_percent"`
	MA5           float64  `json:"ma5"`
	MA10          float64  `json:"ma10"`
	MA20          float64  `json:"ma20"`
	MA60          float64  `json:"ma60"`
	MA5Prev       float64  `json:"ma5_prev"`
	CloseLag3     float64  `json:"close_lag3"`
	MACD          float64  `json:"macd"`
	DIF           float64  `json:"dif"`
	DEA           float64  `json:"dea"`
	DIFPrev       float64  `json:"dif_prev"`
	DEAPrev       float64  `json:"dea_prev"`
	K             float64  `json:"k"`
	J             float64  `json:"j"`
	RSI6          float64  `json:"rsi6"`
	BollUpper     float64  `json:"boll_upper"`
	BollMid       float64  `json:"boll_mid"`
	BollLower     float64  `json:"boll_lower"`
	NetProfit     *float64 `json:"net_profit,omitempty"`
	NetProfitYoy  *float64 `json:"net_profit_yoy,omitempty"`
	RevenueYoy    *float64 `json:"revenue_yoy,omitempty"`
}

type FinalPickResponse struct {
	TradeDate  string      `json:"trade_date"`
	TopN       int         `json:"top_n"`
	Candidates int         `json:"candidates"` // 候选池大小（去重）
	Scored     int         `json:"scored"`     // 实际打分个数（剔除缺关键字段的）
	Picks      []PickStock `json:"picks"`
}

// ---------- 评分函数 ----------

func scoreOne(s presets.RunResult) (int, ScoreBreakdown, bool) {
	var b ScoreBreakdown

	// 1) 板型加权（max 10）—— 创业板 5 / 科创板 5 / 主板 2
	switch s.Market {
	case "创业板":
		b.Board = 5
	case "科创板":
		b.Board = 5
	case "主板":
		b.Board = 2
	default:
		b.Board = 0
	}

	// 2) 技术形态（max 30）
	if s.MA5 > 0 && s.MA10 > 0 && s.MA20 > 0 &&
		s.MA5 > s.MA10 && s.MA10 > s.MA20 {
		b.Technical += 12 // 多头排列
	}
	if s.MA5Prev > 0 && s.MA5 > s.MA5Prev {
		b.Technical += 6 // MA5 上倾
	}
	if s.MACD > 0 {
		b.Technical += 5 // MACD 柱 > 0
	}
	if s.DIFPrev > 0 && s.DEAPrev > 0 && s.DIF > s.DIFPrev && s.DEA > s.DEAPrev &&
		s.DIFPrev <= s.DEAPrev && s.DIF > s.DEA {
		b.Technical += 5 // DIF 上穿 DEA 金叉
	}
	if s.BollUpper > 0 && s.BollMid > 0 && s.Close > 0 &&
		s.Close > s.BollMid && s.Close < s.BollUpper {
		b.Technical += 2 // BOLL 中上轨
	}

	// 3) 动量位置（max 20）—— KDJ K 50-85 + RSI6 60-80
	if s.K >= 50 && s.K <= 85 {
		b.Momentum += 8
	}
	if s.J >= 80 && s.J <= 100 {
		b.Momentum += 5
	}
	if s.RSI6 >= 60 && s.RSI6 <= 80 {
		b.Momentum += 5
	}
	if s.BollMid > 0 && s.Close > 0 && s.Close > s.BollMid && s.Close < s.BollUpper {
		b.Momentum += 2
	}

	// 4) 量价健康（max 18）
	volRatio := 0.0
	if s.Volume > 0 && s.MA5Prev > 0 {
		// 没有 vol_avg5 字段，用近 5 日均量近似 = MA5Prev 反推不准确；这里用 turnover 与 change_percent 间接判断
		// 实际量比通常要 vol_avg5；这里跳过精确计算，用 turnover_rate + change_percent 启发式
		_ = volRatio
	}
	// 用 change_percent + turnover_rate 替代量比
	if s.ChangePercent >= 3 && s.ChangePercent <= 9.8 && s.TurnoverRate >= 5 && s.TurnoverRate <= 15 {
		b.VolumePrice += 12
	} else if s.ChangePercent >= 1 && s.ChangePercent < 3 && s.TurnoverRate >= 3 {
		b.VolumePrice += 8
	} else if s.ChangePercent >= 0 && s.TurnoverRate >= 2 {
		b.VolumePrice += 4
	}
	if s.TurnoverRate >= 3 && s.TurnoverRate <= 12 {
		b.VolumePrice += 4
	}
	if s.ChangePercent >= 3 && s.ChangePercent <= 9.8 {
		b.VolumePrice += 2
	}

	// 5) 成长性（max 12）
	if s.NetProfitYoy != nil {
		v := *s.NetProfitYoy
		switch {
		case v >= 30:
			b.Growth += 6
		case v >= 0:
			b.Growth += 3
		case v < 0:
			b.Penalty -= 5
		}
	}
	if s.RevenueYoy != nil {
		v := *s.RevenueYoy
		switch {
		case v >= 20:
			b.Growth += 4
		case v >= 0:
			b.Growth += 2
		}
	}

	// 6) 资金面（max 10）
	if s.NetAmount > 0 {
		b.Fund += 5
	}
	if s.MfChangePct > 0 {
		b.Fund += 3
	}

	// ---- 减分项 ----
	// 3 日累计涨幅 20~30% 减分（防追高）：close_lag3 是 3 日前收盘
	if s.Close > 0 && s.CloseLag3 > 0 {
		cum3 := (s.Close - s.CloseLag3) / s.CloseLag3 * 100
		switch {
		case cum3 >= 20 && cum3 <= 25:
			b.Penalty -= 3
		case cum3 > 25 && cum3 <= 30:
			b.Penalty -= 6 // >25 应当被预设层排除，兜底再减
		}
	}

	// PE > 200（妖股高估值）
	if s.PETTM > 200 {
		b.Penalty -= 8
	}

	// 当日跌幅 < -3%（弱势不入选）
	if s.ChangePercent < -3 {
		b.Penalty -= 3
	}

	total := b.Board + b.Technical + b.Momentum + b.VolumePrice + b.Growth + b.Fund + b.Penalty
	if total < 0 {
		total = 0
	}
	// 必须至少命中一些"动量 / 技术 / 资金"维度，否则认为是"凑数"的候选
	if b.Technical == 0 && b.Momentum == 0 && b.Fund == 0 {
		return 0, b, false
	}
	return total, b, true
}

// ---------- Handler ----------

func FinalPick(c *gin.Context) {
	var req FinalPickRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.PresetIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "preset_ids 不能为空"})
		return
	}
	if req.TopN <= 0 {
		req.TopN = 2
	}
	if req.TopN > 20 {
		req.TopN = 20
	}

	// 1. 取所有勾选预设的命中（每条 preset 拉到完整命中列表）
	type symbolKey struct {
		sym    string
		source presets.RunResult
	}
	picks := map[string]*PickStock{}
	tradeDate := req.TradeDate

	for _, pid := range req.PresetIDs {
		preset := presets.ByID(pid)
		if preset == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "未找到预设: " + pid})
			return
		}
		// 不分页：拉全量（page=1, pageSize=2000 上限）
		hits, _, err := presets.Run(config.DB, preset.Expression, 1, 2000)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "preset " + pid + " 失败: " + err.Error()})
			return
		}
		if tradeDate == "" && len(hits) > 0 {
			tradeDate = hits[0].TradeDate
		}
		for _, h := range hits {
			if tradeDate != "" && h.TradeDate != tradeDate {
				continue
			}
			if _, ok := picks[h.Symbol]; !ok {
				picks[h.Symbol] = &PickStock{
					Symbol:         h.Symbol,
					Name:           h.Name,
					Industry:       h.Industry,
					Market:         h.Market,
					TradeDate:      h.TradeDate,
					MatchedPresets: []string{pid},
					Close:          h.Close,
					ChangePercent:  h.ChangePercent,
					TurnoverRate:   h.TurnoverRate,
					PETTM:          h.PETTM,
					PB:             h.PB,
					NetAmount:      h.NetAmount,
					MA5:            h.MA5,
					MA10:           h.MA10,
					MA20:           h.MA20,
					MA60:           h.MA60,
					MA5Prev:        h.MA5Prev,
					MACD:           h.MACD,
					DIF:            h.DIF,
					DEA:            h.DEA,
					K:              h.K,
					J:              h.J,
					RSI6:           h.RSI6,
					BollUpper:      h.BollUpper,
					BollMid:        h.BollMid,
					BollLower:      h.BollLower,
					NetProfit:      h.NetProfit,
					NetProfitYoy:   h.NetProfitYoy,
					RevenueYoy:     h.RevenueYoy,
				}
			} else {
				picks[h.Symbol].MatchedPresets = append(picks[h.Symbol].MatchedPresets, pid)
			}
		}
	}

	if tradeDate == "" {
		tradeDate = time.Now().Format("2006-01-02")
	}

	// 2. 关联资金流（mf_change_percent）—— RunResult 没带，从 MV 直接补
	if len(picks) > 0 {
		type mfRow struct {
			Symbol         string
			MfChangePct    float64
		}
		var rows []mfRow
		config.DB.Raw(`
			SELECT symbol, COALESCE(mf_change_percent, 0)
			FROM stock_history_mv
			WHERE trade_date = ?
		`, tradeDate).Scan(&rows)
		mfMap := map[string]float64{}
		for _, r := range rows {
			mfMap[r.Symbol] = r.MfChangePct
		}
		// 关联 DIF/DEA 前一日（用于判金叉）
		var lagRows []struct {
			Symbol    string
			DIFPrev   float64
			DEAPrev   float64
			CloseLag3 float64
		}
		if err := config.DB.Raw(`
			SELECT symbol,
			       COALESCE(dif_lag1, 0)   AS dif_prev,
			       COALESCE(dea_lag1, 0)   AS dea_prev,
			       COALESCE(close_lag3, 0) AS close_lag3
			FROM stock_indicators
			WHERE calc_date = ?
		`, tradeDate).Scan(&lagRows).Error; err != nil {
			fmt.Fprintf(gin.DefaultErrorWriter, "final-pick lag query: %v\n", err)
		}
		lagMap := map[string][3]float64{}
		for _, r := range lagRows {
			lagMap[r.Symbol] = [3]float64{r.DIFPrev, r.DEAPrev, r.CloseLag3}
		}
		for sym, p := range picks {
			p.MfChangePct = mfMap[sym]
			if v, ok := lagMap[sym]; ok {
				p.DIFPrev = v[0]
				p.DEAPrev = v[1]
				p.CloseLag3 = v[2]
			}
		}
	}

	// 3. 评分
	scored := make([]PickStock, 0, len(picks))
	for _, p := range picks {
		// 临时构造 RunResult 用于 scoreOne
		rr := presets.RunResult{
			Symbol:        p.Symbol,
			Name:          p.Name,
			Industry:      p.Industry,
			Market:        p.Market,
			Close:         p.Close,
			ChangePercent: p.ChangePercent,
			TurnoverRate:  p.TurnoverRate,
			PETTM:         p.PETTM,
			PB:            p.PB,
			NetAmount:     p.NetAmount,
			MA5:           p.MA5, MA10: p.MA10, MA20: p.MA20, MA60: p.MA60,
			MA5Prev:       p.MA5Prev,
			MACD:          p.MACD, DIF: p.DIF, DEA: p.DEA,
			DIFPrev:       p.DIFPrev, DEAPrev: p.DEAPrev,
			K:             p.K, J: p.J, RSI6: p.RSI6,
			BollUpper:     p.BollUpper, BollMid: p.BollMid, BollLower: p.BollLower,
			NetProfit:     p.NetProfit, NetProfitYoy: p.NetProfitYoy, RevenueYoy: p.RevenueYoy,
		}
		score, breakdown, ok := scoreOne(rr)
		if !ok {
			continue
		}
		p.Score = score
		p.Breakdown = breakdown
		scored = append(scored, *p)
	}

	// 4. 排序 + 取 TopN（兼顾行业多样性 + 双创优先）
	sort.SliceStable(scored, func(i, j int) bool {
		// 双创 +5 优先：主板 < 创业板 = 科创板
		priority := func(m string) int {
			switch m {
			case "创业板", "科创板":
				return 1
			}
			return 0
		}
		pi, pj := priority(scored[i].Market), priority(scored[j].Market)
		if pi != pj {
			return pi > pj
		}
		return scored[i].Score > scored[j].Score
	})

	// 多样性：TopN 内若 #i 与 #j 同行业则跳过
	final := make([]PickStock, 0, req.TopN)
	usedIndustry := map[string]bool{}
	for _, p := range scored {
		if len(final) >= req.TopN {
			break
		}
		ind := p.Industry
		if ind == "" {
			ind = "_unknown"
		}
		if usedIndustry[ind] {
			continue
		}
		usedIndustry[ind] = true
		final = append(final, p)
	}

	// 如果行业过滤太严导致 < TopN，兜底补齐（允许同行业）
	if len(final) < req.TopN {
		for _, p := range scored {
			if len(final) >= req.TopN {
				break
			}
			// 跳过已在 final 里的
			dup := false
			for _, f := range final {
				if f.Symbol == p.Symbol {
					dup = true
					break
				}
			}
			if dup {
				continue
			}
			final = append(final, p)
		}
	}

	// 4.5) 落库（按 trade_date + symbol upsert，覆盖同日重跑）
		if err := saveFinalPicks(tradeDate, final); err != nil {
			// 不影响返回结果，仅日志
			fmt.Fprintf(gin.DefaultErrorWriter, "saveFinalPicks: %v\n", err)
		}

		c.JSON(http.StatusOK, FinalPickResponse{
		TradeDate:  tradeDate,
		TopN:       req.TopN,
		Candidates: len(picks),
		Scored:     len(scored),
		Picks:      final,
	})
}

var _ = gorm.ErrRecordNotFound // 占位，避免 import 报错


// ---------- 落库 ----------

func saveFinalPicks(tradeDate string, picks []PickStock) error {
	if len(picks) == 0 {
		return nil
	}
	td, err := time.Parse("2006-01-02", tradeDate)
	if err != nil {
		return fmt.Errorf("parse trade_date: %w", err)
	}
	for _, p := range picks {
		bd, _ := json.Marshal(p.Breakdown)
		matched, _ := json.Marshal(p.MatchedPresets)
		row := models.FinalPick{
			TradeDate: td,
			Rank:      0, // 后面再写
			Symbol:    p.Symbol,
			Name:      p.Name,
			Industry:  p.Industry,
			Market:    p.Market,
			Score:     p.Score,
			Breakdown: string(bd),
			Matched:   string(matched),
		}
		// upsert：ON CONFLICT (trade_date, symbol) DO UPDATE
		err := config.DB.Exec(`
			INSERT INTO final_picks
				(trade_date, rank, symbol, name, industry, market, score, breakdown, matched, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?::jsonb, ?, CURRENT_TIMESTAMP)
			ON CONFLICT (trade_date, symbol) DO UPDATE SET
				rank      = EXCLUDED.rank,
				name      = EXCLUDED.name,
				industry  = EXCLUDED.industry,
				market    = EXCLUDED.market,
				score     = EXCLUDED.score,
				breakdown = EXCLUDED.breakdown,
				matched   = EXCLUDED.matched,
				created_at = CURRENT_TIMESTAMP
		`, td, 0, row.Symbol, row.Name, row.Industry, row.Market, row.Score, row.Breakdown, row.Matched).Error
		if err != nil {
			return fmt.Errorf("upsert %s: %w", row.Symbol, err)
		}
	}
	// 回写 rank（按 score DESC）
	for i, p := range picks {
		rank := i + 1
		if err := config.DB.Exec(
			`UPDATE final_picks SET rank = ? WHERE trade_date = ? AND symbol = ?`,
			rank, td, p.Symbol,
		).Error; err != nil {
			return fmt.Errorf("update rank %s: %w", p.Symbol, err)
		}
	}
	return nil
}

// ---------- History 接口 ----------

// FinalPickHistoryItem 历史精选的展示行
type FinalPickHistoryItem struct {
	TradeDate     string         `json:"trade_date"`
	Rank          int            `json:"rank"`
	Symbol        string         `json:"symbol"`
	Name          string         `json:"name"`
	Industry      string         `json:"industry"`
	Market        string         `json:"market"`
	Score         int            `json:"score"`
	Breakdown     ScoreBreakdown `json:"breakdown"`
	MatchedPresets []string      `json:"matched_presets"`
	Close         float64        `json:"close"`
	ChangePercent float64        `json:"change_percent"`
	Volume        float64        `json:"volume"`
	MA5           float64        `json:"ma5"`
	MA10          float64        `json:"ma10"`
	MA5Prev       float64        `json:"ma5_prev"`
}

// FinalPickHistoryResponse 返回 N 天的精选历史（按日期降序）
type FinalPickHistoryResponse struct {
	Days  int                     `json:"days"`
	Total int                     `json:"total"`
	Items []FinalPickHistoryItem  `json:"items"`
}

// FinalPickLatest GET /screen/final-pick/latest
// 返回 final_picks 表里最新交易日的 Top N 缓存结果（不重新计算）
// 用于首页展示：每日数据刷新后预算一次，前端只读
func FinalPickLatest(c *gin.Context) {
	type row struct {
		TradeDate     time.Time
		Rank          int
		Symbol        string
		Name          string
		Industry      string
		Market        string
		Score         int
		Breakdown     string
		Matched       string
		Close         float64
		ChangePercent float64
		Volume        float64
		MA5           float64
		MA10          float64
		MA5Prev       float64
		CreatedAt     time.Time
		NetProfit     *float64
		NetProfitYoy  *float64
		RevenueYoy    *float64
	}
	var rows []row
	err := config.DB.Raw(`
		SELECT fp.trade_date, fp.rank, fp.symbol, fp.name, fp.industry, fp.market,
		       fp.score, fp.breakdown, fp.matched,
		       COALESCE(mv.close, 0)          AS close,
		       COALESCE(mv.change_percent, 0) AS change_percent,
		       COALESCE(mv.volume, 0)         AS volume,
		       COALESCE(mv.ma5, 0)            AS ma5,
		       COALESCE(mv.ma10, 0)           AS ma10,
		       COALESCE(ind.ma5_lag1, 0)      AS ma5_prev,
		       fp.created_at                  AS created_at,
		       fin.net_profit                 AS net_profit,
		       fin.net_profit_yoy             AS net_profit_yoy,
		       fin.revenue_yoy                AS revenue_yoy
		FROM final_picks fp
		LEFT JOIN stock_history_mv mv
		  ON mv.symbol = fp.symbol AND mv.trade_date = fp.trade_date
		LEFT JOIN stock_indicators ind
		  ON ind.symbol = fp.symbol AND ind.calc_date = fp.trade_date
		LEFT JOIN LATERAL (
		    SELECT net_profit, net_profit_yoy, revenue_yoy
		    FROM stock_financial_data
		    WHERE symbol = fp.symbol
		    ORDER BY report_date DESC
		    LIMIT 1
		) fin ON TRUE
		WHERE fp.trade_date = (SELECT MAX(trade_date) FROM final_picks)
		ORDER BY fp.rank ASC
	`).Scan(&rows).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "latest: " + err.Error()})
		return
	}
	if len(rows) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"trade_date":  "",
			"top_n":       0,
			"candidates":  0,
			"scored":      0,
			"picks":       []PickStock{},
			"cached":      false,
			"updated_at":  "",
		})
		return
	}

	items := make([]PickStock, 0, len(rows))
	var tradeDate string
	var updatedAt string
	for _, r := range rows {
		var bd ScoreBreakdown
		_ = json.Unmarshal([]byte(r.Breakdown), &bd)
		var matched []string
		_ = json.Unmarshal([]byte(r.Matched), &matched)
		items = append(items, PickStock{
			Symbol:         r.Symbol,
			Name:           r.Name,
			Industry:       r.Industry,
			Market:         r.Market,
			TradeDate:      r.TradeDate.Format("2006-01-02"),
			Score:          r.Score,
			Breakdown:      bd,
			MatchedPresets: matched,
			Close:          r.Close,
			ChangePercent:  r.ChangePercent,
			MA5:            r.MA5,
			MA10:           r.MA10,
			MA5Prev:        r.MA5Prev,
			NetProfit:      r.NetProfit,
			NetProfitYoy:   r.NetProfitYoy,
			RevenueYoy:     r.RevenueYoy,
		})
		tradeDate = r.TradeDate.Format("2006-01-02")
		updatedAt = r.CreatedAt.Format("2006-01-02 15:04:05")
	}

	c.JSON(http.StatusOK, gin.H{
		"trade_date": tradeDate,
		"top_n":      len(items),
		"candidates": len(items),
		"scored":     len(items),
		"picks":      items,
		"cached":     true,
		"updated_at": updatedAt,
	})
}

// FinalPickHistory GET /screen/final-pick/history?days=7
func FinalPickHistory(c *gin.Context) {
	days := 7
	if v := c.Query("days"); v != "" {
		if d, err := strconv.Atoi(v); err == nil && d > 0 && d <= 60 {
			days = d
		}
	}
	type row struct {
		TradeDate     time.Time
		Rank          int
		Symbol        string
		Name          string
		Industry      string
		Market        string
		Score         int
		Breakdown     string
		Matched       string
		Close         float64
		ChangePercent float64
		Volume        float64
		MA5           float64
		MA10          float64
		MA5Prev       float64
	}
	var rows []row
	err := config.DB.Raw(`
		SELECT fp.trade_date, fp.rank, fp.symbol, fp.name, fp.industry, fp.market,
		       fp.score, fp.breakdown, fp.matched,
		       COALESCE(mv.close, 0)          AS close,
		       COALESCE(mv.change_percent, 0) AS change_percent,
		       COALESCE(mv.volume, 0)         AS volume,
		       COALESCE(mv.ma5, 0)            AS ma5,
		       COALESCE(mv.ma10, 0)           AS ma10,
		       COALESCE(ind.ma5_lag1, 0)      AS ma5_prev
		FROM final_picks fp
		LEFT JOIN stock_history_mv mv
		  ON mv.symbol = fp.symbol AND mv.trade_date = fp.trade_date
		LEFT JOIN stock_indicators ind
		  ON ind.symbol = fp.symbol AND ind.calc_date = fp.trade_date
		WHERE fp.trade_date >= (CURRENT_DATE - (? || ' days')::interval)
		ORDER BY fp.trade_date DESC, fp.rank ASC
	`, strconv.Itoa(days)).Scan(&rows).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "history: " + err.Error()})
		return
	}
	items := make([]FinalPickHistoryItem, 0, len(rows))
	for _, r := range rows {
		var bd ScoreBreakdown
		_ = json.Unmarshal([]byte(r.Breakdown), &bd)
		var matched []string
		_ = json.Unmarshal([]byte(r.Matched), &matched)
		items = append(items, FinalPickHistoryItem{
			TradeDate:     r.TradeDate.Format("2006-01-02"),
			Rank:          r.Rank,
			Symbol:        r.Symbol,
			Name:          r.Name,
			Industry:      r.Industry,
			Market:        r.Market,
			Score:         r.Score,
			Breakdown:     bd,
			MatchedPresets: matched,
			Close:         r.Close,
			ChangePercent: r.ChangePercent,
			Volume:        r.Volume,
			MA5:           r.MA5,
			MA10:          r.MA10,
			MA5Prev:       r.MA5Prev,
		})
	}
	c.JSON(http.StatusOK, FinalPickHistoryResponse{
		Days:  days,
		Total: len(items),
		Items: items,
	})
}

var _ = gorm.ErrRecordNotFound // 占位，避免 import 报错
