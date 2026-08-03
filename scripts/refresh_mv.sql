-- ============================================================
-- 创建 / 刷新 stock_history_mv 物化视图
-- 把 stock_daily_data + stock_indicators + stock_money_flow_all 三表对齐
-- 由 scripts/refresh_mv.py 自动调用（也可手动 psql 执行）
-- ============================================================

-- 1) 删除旧的 MV（如果有）
DROP MATERIALIZED VIEW IF EXISTS stock_history_mv;

-- 2) 创建 MV
CREATE MATERIALIZED VIEW stock_history_mv AS
SELECT
    d.symbol::varchar(10)                         AS symbol,
    bi.name::varchar(50)                          AS name,
    bi.industry::varchar(50)                      AS industry,
    bi.market::varchar(20)                        AS market,
    d.trade_date                                  AS trade_date,
    d.open, d.high, d.low, d.close,
    d.volume, d.turnover,
    d.change_percent, d.change_amount, d.turnover_rate,
    d.pe_ttm, d.pb, d.amplitude,
    i.ma5, i.ma10, i.ma20, i.ma60,
    i.macd, i.dif, i.dea,
    i.k, i.d, i.j,
    i.rsi6, i.rsi12, i.rsi24,
    i.boll_upper, i.boll_mid, i.boll_lower,
    mf.inflow_amount  AS in_amount,
    mf.outflow_amount AS out_amount,
    mf.net_amount     AS net_amount,
    mf.turnover       AS mf_turnover,
    mf.change_percent AS mf_change_percent
FROM stock_daily_data d
LEFT JOIN stock_basic_info        bi ON bi.symbol = d.symbol
LEFT JOIN stock_indicators        i  ON i.symbol = d.symbol  AND i.calc_date = d.trade_date
LEFT JOIN stock_money_flow_all    mf ON mf.symbol = d.symbol AND mf.trade_date = d.trade_date AND mf.time_span = 0;

-- 3) 唯一索引（CONCURRENTLY 刷新的前提）
CREATE UNIQUE INDEX IF NOT EXISTS uk_stock_history_mv
    ON stock_history_mv(symbol, trade_date);

CREATE INDEX IF NOT EXISTS idx_stock_history_mv_symbol_date
    ON stock_history_mv(symbol, trade_date DESC);

CREATE INDEX IF NOT EXISTS idx_stock_history_mv_change
    ON stock_history_mv(change_percent DESC);

-- 4) 首次填充（同步）
REFRESH MATERIALIZED VIEW stock_history_mv;
