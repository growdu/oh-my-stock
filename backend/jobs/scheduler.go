package jobs

import (
	"context"
	"fmt"
	"log"
	"oh-my-stock/config"
	"oh-my-stock/notify"
	"sync"
	"time"

	"oh-my-stock/fetcher"
	"oh-my-stock/models"
)

// Start 启动后开始：1) 异步拉全量列表；2) 周期抓取近期日 K；3) 周期裁剪。
func Start(ctx context.Context) {
	go runOnce(ctx)

	go loop(ctx, 5*time.Minute, func(ctx context.Context) {
		runIncrementalFetch(ctx)
		if n, err := fetcher.PurgeOldDaily(); err == nil {
			log.Printf("✅ 裁剪 stock_daily_data：删除 %d 行（>30 天）", n)
		} else {
			log.Printf("⚠️ 裁剪 stock_daily_data 失败: %v", err)
		}
		if n, err := fetcher.PurgeOldHistoryMV(); err == nil {
			log.Printf("✅ 裁剪 stock_history_mv：删除 %d 行（>30 天）", n)
		} else {
			log.Printf("⚠️ 裁剪 stock_history_mv 失败: %v", err)
		}
		if n, err := fetcher.PurgeOldMoneyFlowDaily(); err == nil {
			log.Printf("✅ 裁剪 stock_money_flow_daily：删除 %d 行（>30 天）", n)
		} else {
			log.Printf("⚠️ 裁剪 stock_money_flow_daily 失败: %v", err)
		}
	})

	// 规则触发通知：每 5 分钟一次，与数据抓取同步。
	// 实际只依赖最新一个 trade_date 的快照，去重粒度是 (user, rule, symbol, 当日)，不会重复刷屏。
	go loop(ctx, 5*time.Minute, func(ctx context.Context) {
		runRuleChecks(ctx)
	})
}

// runRuleChecks 对所有用户跑一次规则匹配，写入新通知。
// 失败计入日志但不阻塞下次循环。
func runRuleChecks(ctx context.Context) {
	n, err := notify.RunForAllUsers(config.DB)
	if err != nil {
		log.Printf("⚠️ 规则匹配失败: %v", err)
		return
	}
	if n > 0 {
		log.Printf("✅ 规则匹配：写入 %d 条新通知", n)
	} else {
		log.Printf("ℹ️ 规则匹配：无新增通知")
	}
}

// runOnce 启动时执行一次：检测表是否为空，必要时拉全量列表
func runOnce(ctx context.Context) {
	cnt := fetcher.CountBasicInfo()
	if cnt > 0 {
		log.Printf("ℹ️ stock_basic_info 已有 %d 行，跳过全量拉取", cnt)
		return
	}
	log.Printf("⏳ stock_basic_info 为空，开始从东方财富拉全量列表...")
	if err := fetchAndPersistStockList(ctx); err != nil {
		log.Printf("❌ 全量列表拉取失败: %v", err)
		return
	}
	log.Printf("✅ stock_basic_info 初始化完成")
}

func fetchAndPersistStockList(ctx context.Context) error {
	items, err := fetcher.FetchSinaList(ctx)
	if err != nil {
		return err
	}
	log.Printf("✅ 已从新浪行情中心获取 %d 行", len(items))
	const chunk = 300
	total := 0
	for i := 0; i < len(items); i += chunk {
		end := i + chunk
		if end > len(items) {
			end = len(items)
		}
		rows := make([]models.StockBasicInfo, 0, end-i)
		for _, it := range items[i:end] {
			if len(it.Code) < 6 {
				continue
			}
			rows = append(rows, models.StockBasicInfo{
				Symbol: it.Code,
				Name:   it.Name,
				Status: "上市",
			})
		}
		n, err := fetcher.UpsertBasicInfo(rows)
		if err != nil {
			return err
		}
		total += n
		log.Printf("✅ 写入 stock_basic_info %d/%d（chunk %d 条）", total, len(items), n)
	}
	log.Printf("✅ stock_basic_info 全量入库完成 %d 行", total)
	log.Printf("⏳ 启动后端东财 detail 补全 industry/market/area...")
	bumpIndustry := RefetchStockBasics(ctx, fetcherListAllSymbols())
	if bumpIndustry.Updated > 0 {
		log.Printf("✅ industry/market/area 补全完成 %d 行（失败 %d）", bumpIndustry.Updated, len(bumpIndustry.Failed))
	} else {
		log.Printf("⚠️ industry/market/area 补全 0 行（%d 失败）", len(bumpIndustry.Failed))
	}
	return nil
}

// RefetchSummary 用于 admin 控制器返回
type RefetchSummary struct {
	Updated int
	Failed  []string
}

// fetcherListAllSymbols 拉所有 symbol（包装 fetcher.ListAllSymbols 统一调用）
func fetcherListAllSymbols() []string {
	return fetcher.ListAllSymbols()
}

// RefetchStockBasics 同步补全指定 symbols 的 industry/market/area/pe/pb/listing_date
// 一次性拉全市场估值表，然后按 symbol 索引，避免每只请求 1 次。
func RefetchStockBasics(ctx context.Context, symbols []string) RefetchSummary {
	summary := RefetchSummary{}
	if len(symbols) == 0 {
		return summary
	}
	// 1) 一次拉全市场估值表
	t0 := time.Now()
	vrows, err := fetcher.FetchValuationAll(ctx)
	if err != nil {
		return summary
	}
	index := make(map[string]fetcher.ValuationRow, len(vrows))
	for _, r := range vrows {
		index[r.Symbol] = r
	}
	log.Printf("✅ 拉一次性估值表 %d 行（%s）", len(vrows), time.Since(t0))

	// 2) 并发拉 detail（name/industry/area/total_shares）并按 symbol 合并估值
	type result struct {
		sym string
		row models.StockBasicInfo
	}
	sem := make(chan struct{}, 10)
	results := make(chan result, len(symbols))
	var wg sync.WaitGroup
	for _, sym := range symbols {
		sym := sym
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			detail, err := fetcher.FetchEastMoneyDetail(ctx, sym)
			if err != nil {
				log.Printf("⚠️ %s 拉 detail 失败: %v", sym, err)
				summary.Failed = append(summary.Failed, sym)
				return
			}
			row := models.StockBasicInfo{
				Symbol:      detail.Symbol,
				Name:        fallbackName(detail.Name, sym),
				Industry:    detail.Industry,
				Area:        detail.Area,
				Market:      detail.Market,
				Status:      "上市",
				TotalShares: detail.TotalShares,
			}
			if v, ok := index[sym]; ok {
				row.PETTM = v.PETTM
				row.PB = v.PB
				if v.ListingDate != "" {
					t, err := time.Parse("2006-01-02", v.ListingDate)
					if err == nil {
						row.ListingDate = &t
					}
				}
			}
			results <- result{sym: sym, row: row}
		}()
	}
	wg.Wait()
	close(results)

	const chunk = 200
	batch := make([]models.StockBasicInfo, 0, chunk)
	for r := range results {
		batch = append(batch, r.row)
		if len(batch) >= chunk {
			n, err := fetcher.UpsertBasicInfoWithValuation(batch)
			if err != nil {
				log.Printf("⚠️ batch upsert: %v", err)
			}
			summary.Updated += n
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		n, err := fetcher.UpsertBasicInfoWithValuation(batch)
		if err != nil {
			log.Printf("⚠️ tail batch upsert: %v", err)
		}
		summary.Updated += n
	}
	return summary
}

func fallbackName(detailName, sym string) string {
	if detailName != "" {
		return detailName
	}
	return sym
}

// RefetchStockBasicsAllFull 基于 DB 现有 symbols，全部一次性补全 industry/market/area。
func RefetchStockBasicsAllFull(ctx context.Context) error {
	symbols := fetcher.ListAllSymbols()
	if len(symbols) == 0 {
		log.Printf("ℹ️ DB 里没有 symbol 可补全")
		return nil
	}
	log.Printf("⏳ admin 全量补全 %d symbols 的 industry/market/area", len(symbols))
	summary := RefetchStockBasics(ctx, symbols)
	log.Printf("✅ admin 全量补全 完成 %d（失败 %d）", summary.Updated, len(summary.Failed))
	return nil
}

// RefetchStockDailyAll 触发全量最近 7 天日 K 抓取
func RefetchStockDailyAll(ctx context.Context) {
	symbols := fetcher.ListAllSymbols()
	if len(symbols) == 0 {
		log.Printf("ℹ️ 没有 symbol 可抓取")
		return
	}
	log.Printf("⏳ admin 触发全量日 K 抓取 %d 只", len(symbols))
	fetchWithLimit(ctx, symbols, 10, 7)
	log.Printf("✅ admin 全量日 K 抓取完成")
}

// runIncrementalFetch 增量：仅拉最近 24h 内有日 K 的 symbol
func runIncrementalFetch(ctx context.Context) {
	symbols := fetcher.ActiveSymbolsSince(time.Now().Add(-24 * time.Hour))
	if len(symbols) == 0 {
		// 兜底：拉所有 symbol 的最近 1 条（首次启动后还可能没数据）
		symbols = fetcher.ListAllSymbols()
	}
	if len(symbols) == 0 {
		log.Printf("ℹ️ 没有 symbol 需要抓取")
		return
	}
	log.Printf("⏳ 增量抓取 %d 只股票最近 7 天日 K...", len(symbols))
	fetchWithLimit(ctx, symbols, 10, 7)
	log.Printf("✅ 增量抓取完成")
}

func loop(ctx context.Context, interval time.Duration, fn func(ctx context.Context)) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			fn(ctx)
		}
	}
}

// fetchWithLimit 并发抓取，limit 控制最大并发数
func fetchWithLimit(ctx context.Context, symbols []string, limit int, days int) {
	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup
	for _, sym := range symbols {
		wg.Add(1)
		sem <- struct{}{}
		go func(s string) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := fetchOneSymbol(ctx, s, days); err != nil {
				log.Printf("⚠️ %s 抓取失败: %v", s, err)
			}
		}(sym)
	}
	wg.Wait()
}

// fetchOneSymbol 拉一只，写库：日K + 资金流 + 技术指标
func fetchOneSymbol(ctx context.Context, symbol string, days int) error {
	rows, err := fetcher.FetchRecentDaily(ctx, symbol, days)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	prepared := make([]models.StockDailyData, 0, len(rows))
	for i, r := range rows {
		t, perr := time.Parse("2006-01-02", r.Day)
		if perr != nil {
			continue
		}
		entry := models.StockDailyData{
			Symbol:    symbol,
			TradeDate: t,
			Open:      r.Open,
			High:      r.High,
			Low:       r.Low,
			Close:     r.Close,
			Volume:    int64(r.Volume),
			Turnover:  r.Turnover,
		}
		if i > 0 {
			prev := rows[i-1].Close
			if prev != 0 {
				entry.ChangeAmount = fetcher.Round4(r.Close - prev)
				entry.ChangePercent = fetcher.Round4((r.Close - prev) / prev * 100)
			}
		}
		prepared = append(prepared, entry)
	}
	if n, err := fetcher.UpsertDaily(prepared); err != nil {
		return fmt.Errorf("upsert: %w", err)
	} else {
		log.Printf("✅ %s 写入 daily %d 行", symbol, n)
	}

	// 资金流（best-effort，失败不影响主流程）
	flows, ferr := fetcher.FetchEastMoneyFlowDays(ctx, symbol, days)
	if ferr != nil {
		log.Printf("⚠️ %s 拉资金流失败: %v", symbol, ferr)
	} else if n, ferr := fetcher.UpsertMoneyFlowDaily(flows); ferr != nil {
		log.Printf("⚠️ %s 写资金流失败: %v", symbol, ferr)
	} else {
		log.Printf("✅ %s 写入资金流 %d 行", symbol, n)
	}

	// 历史 MV（资金流已通过表关联在 UpsertHistoryMV 内合并）
	if n, err := fetcher.UpsertHistoryMV(prepared); err != nil {
		log.Printf("⚠️ %s 同步到 stock_history_mv: %v", symbol, err)
	} else {
		log.Printf("✅ %s 写入 mv %d 行", symbol, n)
	}

	// 技术指标（取最近 60 天日 K 计算）
	recent, err := fetcher.LoadRecentDaily(symbol, 90)
	if err == nil && len(recent) >= 30 {
		inds := fetcher.ComputeIndicators(symbol, recent)
		if n, err := fetcher.UpsertIndicators(inds); err != nil {
			log.Printf("⚠️ %s 写技术指标失败: %v", symbol, err)
		} else {
			log.Printf("✅ %s 写入指标 %d 行", symbol, n)
		}
	}
	return nil
}

