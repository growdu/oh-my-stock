package presets

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCompile_Field(t *testing.T) {
	c := json.RawMessage(`{"all":[{"type":"field","name":"close","op":"gt","value":5}]}`)
	r, err := Compile(c)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.Where, "latest.close > $1") {
		t.Errorf("where = %q", r.Where)
	}
	if len(r.Args) != 1 || r.Args[0] != float64(5) {
		t.Errorf("args = %v", r.Args)
	}
}

func TestCompile_MAAlignment(t *testing.T) {
	c := json.RawMessage(`{"all":[{"type":"ma_alignment","order":["ma5","ma10","ma20","ma60"]}]}`)
	r, err := Compile(c)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"latest.ma5 > latest.ma10", "latest.ma10 > latest.ma20", "latest.ma20 > latest.ma60"}
	for _, w := range want {
		if !strings.Contains(r.Where, w) {
			t.Errorf("missing %q in %q", w, r.Where)
		}
	}
}

func TestCompile_YangStreak(t *testing.T) {
	c := json.RawMessage(`{"all":[{"type":"yang_streak","days":3}]}`)
	r, err := Compile(c)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		want := "yang_lag" + []string{"0", "1", "2", "3"}[i] + " = TRUE"
		if !strings.Contains(r.Where, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestCompile_Exclude(t *testing.T) {
	c := json.RawMessage(`{"all":[{"type":"field","name":"close","op":"gt","value":5}],"exclude":[{"type":"is_st"}]}`)
	r, err := Compile(c)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.Where, "NOT ((latest.name LIKE '%ST%' OR latest.name LIKE '%st%'))") {
		t.Errorf("where = %q", r.Where)
	}
}

func TestCompile_VolumeIncreasing(t *testing.T) {
	c := json.RawMessage(`{"all":[{"type":"volume_increasing","days":3,"min_ratio":1.2}]}`)
	r, err := Compile(c)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"vol_lag1", "vol_lag2", "vol_lag3", "$1"} {
		if !strings.Contains(r.Where, want) {
			t.Errorf("missing %q in %q", want, r.Where)
		}
	}
	if r.Args[0] != float64(1.2) {
		t.Errorf("ratio arg = %v", r.Args)
	}
}

func TestCompile_BollPosition(t *testing.T) {
	c := json.RawMessage(`{"all":[{"type":"boll_position","position":"lower"}]}`)
	r, err := Compile(c)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.Where, "latest.close <= latest.boll_lower") {
		t.Errorf("where = %q", r.Where)
	}
}

func TestCompile_BadField(t *testing.T) {
	c := json.RawMessage(`{"all":[{"type":"field","name":"bogus","op":"gt","value":1}]}`)
	if _, err := Compile(c); err == nil {
		t.Error("expected error for unknown field")
	}
}

func TestCompile_BadType(t *testing.T) {
	c := json.RawMessage(`{"all":[{"type":"x"}]}`)
	if _, err := Compile(c); err == nil {
		t.Error("expected error for unknown type")
	}
}

func TestCompile_MaCross(t *testing.T) {
	c := json.RawMessage(`{"all":[{"type":"ma_cross","fast":"ma5","slow":"ma10","direction":"golden"}]}`)
	r, err := Compile(c)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"latest.ma5_lag1 IS NOT NULL", "latest.ma10_lag1 IS NOT NULL",
		"latest.ma5 IS NOT NULL", "latest.ma10 IS NOT NULL",
		"latest.ma5_lag1 <= latest.ma10_lag1",
		"latest.ma5 > latest.ma10",
	} {
		if !strings.Contains(r.Where, want) {
			t.Errorf("missing %q in %q", want, r.Where)
		}
	}
}

func TestCompile_MaCrossDeath(t *testing.T) {
	c := json.RawMessage(`{"all":[{"type":"ma_cross","fast":"ma5","slow":"ma20","direction":"death"}]}`)
	r, err := Compile(c)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.Where, "latest.ma5_lag1 >= latest.ma20_lag1") ||
		!strings.Contains(r.Where, "latest.ma5 < latest.ma20") {
		t.Errorf("death cross sql wrong: %q", r.Where)
	}
}

func TestCompile_YinStreak(t *testing.T) {
	c := json.RawMessage(`{"all":[{"type":"yin_streak","days":3}]}`)
	r, err := Compile(c)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"yang_lag0 = FALSE", "yang_lag1 = FALSE", "yang_lag2 = FALSE",
	} {
		if !strings.Contains(r.Where, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestCompile_ChangePercentRange(t *testing.T) {
	c := json.RawMessage(`{"all":[{"type":"change_percent_range","min":-3,"max":3}]}`)
	r, err := Compile(c)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.Where, "latest.change_percent BETWEEN $1 AND $2") {
		t.Errorf("where = %q", r.Where)
	}
	if r.Args[0] != float64(-3) || r.Args[1] != float64(3) {
		t.Errorf("args = %v", r.Args)
	}
}

func TestCompile_IndustryIn(t *testing.T) {
	c := json.RawMessage(`{"all":[{"type":"industry_in","industries":["半导体","银行"]}]}`)
	r, err := Compile(c)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.Where, "basic.industry = $1") ||
		!strings.Contains(r.Where, "basic.industry = $2") {
		t.Errorf("where = %q", r.Where)
	}
	if r.Args[0] != "半导体" || r.Args[1] != "银行" {
		t.Errorf("args = %v", r.Args)
	}
}

func TestCompile_Bias(t *testing.T) {
	c := json.RawMessage(`{"all":[{"type":"bias","ma":"ma20","min":-5,"max":5}]}`)
	r, err := Compile(c)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.Where, "(latest.close - latest.ma20) / latest.ma20 * 100") {
		t.Errorf("where = %q", r.Where)
	}
}

func TestCompile_LimitUp(t *testing.T) {
	c := json.RawMessage(`{"all":[{"type":"limit_up","min_pct":9.8}]}`)
	r, err := Compile(c)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.Where, "latest.change_percent >= $1") {
		t.Errorf("where = %q", r.Where)
	}
	if r.Args[0] != 9.8 {
		t.Errorf("args = %v", r.Args)
	}
}

func TestCompile_LimitUpDefault(t *testing.T) {
	c := json.RawMessage(`{"all":[{"type":"limit_up"}]}`)
	r, err := Compile(c)
	if err != nil {
		t.Fatal(err)
	}
	if r.Args[0] != 9.5 {
		t.Errorf("default min_pct should be 9.5, got %v", r.Args[0])
	}
}

func TestCompile_LimitDown(t *testing.T) {
	c := json.RawMessage(`{"all":[{"type":"limit_down","max_pct":-9.8}]}`)
	r, err := Compile(c)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.Where, "latest.change_percent <= $1") {
		t.Errorf("where = %q", r.Where)
	}
}

func TestCompile_BollPctB(t *testing.T) {
	c := json.RawMessage(`{"all":[{"type":"boll_pct_b","min":0,"max":0.2}]}`)
	r, err := Compile(c)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.Where, "latest.boll_upper > latest.boll_lower") {
		t.Errorf("where = %q", r.Where)
	}
	if !strings.Contains(r.Where, "(latest.close - latest.boll_lower) / (latest.boll_upper - latest.boll_lower)") {
		t.Errorf("where = %q", r.Where)
	}
}

func TestCompile_BollWidth(t *testing.T) {
	c := json.RawMessage(`{"all":[{"type":"boll_width","min":0.05,"max":0.5}]}`)
	r, err := Compile(c)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.Where, "(latest.boll_upper - latest.boll_lower) / latest.boll_mid") {
		t.Errorf("where = %q", r.Where)
	}
}

func TestCompile_MacdHistogram(t *testing.T) {
	c := json.RawMessage(`{"all":[{"type":"macd_histogram","sign":"positive","growing":true}]}`)
	r, err := Compile(c)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"latest.dif IS NOT NULL", "latest.dea IS NOT NULL",
		"(latest.dif - latest.dea) * 2 > 0",
		"ABS(latest.dif - latest.dea) > ABS(latest.dif_lag1 - latest.dea_lag1)",
	} {
		if !strings.Contains(r.Where, want) {
			t.Errorf("missing %q in %q", want, r.Where)
		}
	}
}

func TestCompile_LowBreakout(t *testing.T) {
	c := json.RawMessage(`{"all":[{"type":"low_breakout","lookback":20}]}`)
	r, err := Compile(c)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.Where, "latest.low_min20 IS NOT NULL") ||
		!strings.Contains(r.Where, "latest.close < latest.low_min20") {
		t.Errorf("where = %q", r.Where)
	}
}

func TestCompile_ClosePosition(t *testing.T) {
	c := json.RawMessage(`{"all":[{"type":"close_position","lookback":60,"min":0.5,"max":1.0}]}`)
	r, err := Compile(c)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.Where, "latest.low_min60") ||
		!strings.Contains(r.Where, "latest.high_max60") ||
		!strings.Contains(r.Where, "latest.high_max60 > latest.low_min60") {
		t.Errorf("where = %q", r.Where)
	}
}

func TestCompile_AmountRange(t *testing.T) {
	c := json.RawMessage(`{"all":[{"type":"amount_range","min":100000000,"max":10000000000}]}`)
	r, err := Compile(c)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.Where, "latest.volume * latest.close BETWEEN $1 AND $2") {
		t.Errorf("where = %q", r.Where)
	}
}

func TestCompile_VolumeRatioMax(t *testing.T) {
	c := json.RawMessage(`{"all":[{"type":"volume_ratio","max":0.8}]}`)
	r, err := Compile(c)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.Where, "latest.volume / latest.vol_avg5 <= $1") {
		t.Errorf("where = %q", r.Where)
	}
}

func TestCompile_VolumeRatioBoth(t *testing.T) {
	c := json.RawMessage(`{"all":[{"type":"volume_ratio","min":0.5,"max":1.2}]}`)
	r, err := Compile(c)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.Where, "BETWEEN $1 AND $2") {
		t.Errorf("where = %q", r.Where)
	}
}

func TestByID(t *testing.T) {
	for _, p := range All {
		if ByID(p.ID) == nil {
			t.Errorf("ByID(%s) returned nil", p.ID)
		}
	}
	if ByID("nope") != nil {
		t.Error("ByID unknown should be nil")
	}
}

// 空表达式应报错，避免上层误以为永远命中。
func TestCompile_Empty(t *testing.T) {
	if _, err := Compile([]byte("")); err == nil {
		t.Fatal("expected error for empty expression")
	}
	if _, err := Compile([]byte("{}")); err == nil {
		t.Fatal("expected error for empty all/exclude")
	}
}

// 全部内置预设都能成功编译；任何一个 type 写错就 panic 在生产。
func TestAllPresets_Compile(t *testing.T) {
	for _, p := range All {
		exprJSON, err := json.Marshal(p.Expression)
		if err != nil {
			t.Fatalf("%s marshal: %v", p.ID, err)
		}
		if _, err := Compile(exprJSON); err != nil {
			t.Errorf("preset %s compile failed: %v", p.ID, err)
		}
	}
}

// 校验内置预设里不会出现未知 type，否则用户在前端跑预设会直接失败。
func TestAllPresets_NoUnknownTypes(t *testing.T) {
	typeSet := map[string]struct{}{
		"field": {}, "field_between": {}, "ma_compare": {}, "ma_alignment": {},
		"ma_slope": {}, "ma_cross": {}, "volume_ratio": {}, "volume_increasing": {},
		"window_field": {}, "yang_streak": {}, "yin_streak": {}, "yang_then_yin": {},
		"cumulative_change": {},
		"breakout_high":     {}, "low_breakout": {}, "close_position": {},
		"macd_cross": {}, "kdj_cross": {}, "rsi_range": {}, "close_vs_ma": {},
		"boll_position": {}, "boll_pct_b": {}, "boll_width": {}, "bias": {},
		"macd_histogram":       {},
		"change_percent_range": {}, "turnover_rate_range": {}, "amount_range": {},
		"limit_up": {}, "limit_down": {},
		"is_st": {}, "is_not_st": {},
		"list_age_days_gte": {}, "list_age_days_lt": {}, "market_cap_yi": {},
		"industry_in": {}, "industry_not_in": {},
	}
	var walk func(v interface{})
	walk = func(v interface{}) {
		switch v := v.(type) {
		case []interface{}:
			for _, x := range v {
				walk(x)
			}
		case map[string]interface{}:
			if typ, ok := v["type"].(string); ok {
				if _, known := typeSet[typ]; !known {
					t.Errorf("unknown condition type %q", typ)
				}
			}
			for _, x := range v {
				walk(x)
			}
		}
	}
	for _, p := range All {
		walk(p.Expression)
	}
}
