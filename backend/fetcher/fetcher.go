package fetcher

import (
	"context"
	"log"
	"strings"
	"time"
)

// MarketFromSymbol 根据 6 位代码判断上交所/深交所/北交所前缀
func MarketFromSymbol(code string) string {
	switch {
	case strings.HasPrefix(code, "60"), strings.HasPrefix(code, "68"), strings.HasPrefix(code, "9"):
		return "sh"
	case strings.HasPrefix(code, "8"), strings.HasPrefix(code, "43"):
		return "bj"
	default:
		return "sz"
	}
}

// FetchRecentDaily 拉取某只股票最近 days 天的日 K 线（不前复权）。
// 调用方负责写入 DB：返回值即原始 K 线。
func FetchRecentDaily(ctx context.Context, rawSymbol string, days int) ([]SinaDaily, error) {
	sym := strings.ToLower(MarketFromSymbol(rawSymbol) + rawSymbol)
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		rows, err := FetchSinaDaily(ctx, sym, days)
		if err == nil {
			return rows, nil
		}
		lastErr = err
		log.Printf("⚠️ sina 重试第 %d 次 %s: %v", attempt+1, rawSymbol, err)
		time.Sleep(500 * time.Millisecond)
	}
	return nil, lastErr
}

// Round4 四舍五入到 4 位小数（保留财务字段精度）。
// 多个 caller（scheduler / controller）共享，避免重复实现走偏。
func Round4(v float64) float64 {
	return float64(int64(v*10000+0.5)) / 10000
}
