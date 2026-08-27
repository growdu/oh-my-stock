package controllers

import (
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"oh-my-stock/config"
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
	// 3 日累计涨幅 20~25%：已经在 commonExcludes 排掉 >25%；20~25 是 -3
	if s.MA5Prev > 0 && s.MA10Prev > 0 { // 仅作占位：3 日涨幅在 MV 里有 close_lag3 但 RunResult 未带
		// 暂时跳过精确计算；后续如要加需扩展 RunResult 加 close_lag3
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
			Symbol  string
			DIFPrev float64
			DEAPrev float64
		}
		config.DB.Raw(`
			SELECT symbol, dif_lag1, dea_lag1
			FROM stock_history_mv
			WHERE trade_date = ?
		`, tradeDate).Scan(&lagRows)
		lagMap := map[string][2]float64{}
		for _, r := range lagRows {
			lagMap[r.Symbol] = [2]float64{r.DIFPrev, r.DEAPrev}
		}
		for sym, p := range picks {
			p.MfChangePct = mfMap[sym]
			if v, ok := lagMap[sym]; ok {
				p.DIFPrev = v[0]
				p.DEAPrev = v[1]
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
			NetProfitYoy:  p.NetProfitYoy, RevenueYoy: p.RevenueYoy,
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

	c.JSON(http.StatusOK, FinalPickResponse{
		TradeDate:  tradeDate,
		TopN:       req.TopN,
		Candidates: len(picks),
		Scored:     len(scored),
		Picks:      final,
	})
}

var _ = gorm.ErrRecordNotFound // 占位，避免 import 报错
