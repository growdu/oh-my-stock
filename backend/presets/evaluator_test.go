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
		want := "yang_lag" + []string{"0","1","2","3"}[i] + " = TRUE"
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
		"ma_slope": {}, "volume_ratio": {}, "volume_increasing": {},
		"window_field": {}, "yang_streak": {}, "cumulative_change": {},
		"breakout_high": {}, "macd_cross": {}, "kdj_cross": {}, "rsi_range": {}, "close_vs_ma": {},
		"boll_position": {}, "is_st": {}, "is_not_st": {},
		"list_age_days_gte": {}, "list_age_days_lt": {}, "market_cap_yi": {},
	}
	var walk func(v interface{})
	walk = func(v interface{}) {
		switch v := v.(type) {
		case []interface{}:
			for _, x := range v { walk(x) }
		case map[string]interface{}:
			if typ, ok := v["type"].(string); ok {
				if _, known := typeSet[typ]; !known {
					t.Errorf("unknown condition type %q", typ)
				}
			}
			for _, x := range v { walk(x) }
		}
	}
	for _, p := range All {
		walk(p.Expression)
	}
}
