-- ============================================================
-- oh-my-stock · 全量建表 DDL（幂等）
-- 用法：psql "$DATABASE_URL" -f scripts/create_table.sql
-- 创建扩展 pgcrypto 用于 gen_random_uuid()
-- ============================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ============================================================
-- 1. 股票基础信息表
-- ============================================================
CREATE TABLE IF NOT EXISTS stock_basic_info (
    id                  SERIAL PRIMARY KEY,
    symbol              VARCHAR(10)  NOT NULL UNIQUE,
    name                VARCHAR(50)  NOT NULL,
    full_name           VARCHAR(100),
    industry            VARCHAR(50),
    area                VARCHAR(50),
    market              VARCHAR(20),
    listing_date        DATE,
    outstanding_shares  DECIMAL(20,4),
    total_shares        DECIMAL(20,4),
    is_hs               BOOLEAN,
    status              VARCHAR(20),
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_stock_basic_symbol   ON stock_basic_info(symbol);
CREATE INDEX IF NOT EXISTS idx_stock_basic_industry ON stock_basic_info(industry);
CREATE INDEX IF NOT EXISTS idx_stock_basic_market   ON stock_basic_info(market);

-- ============================================================
-- 2. 股票日线数据
-- ============================================================
CREATE TABLE IF NOT EXISTS stock_daily_data (
    id               SERIAL PRIMARY KEY,
    symbol           VARCHAR(10) NOT NULL,
    trade_date       DATE        NOT NULL,
    open             DECIMAL(12,4),
    high             DECIMAL(12,4),
    low              DECIMAL(12,4),
    close            DECIMAL(12,4),
    adj_close        DECIMAL(12,4),
    volume           BIGINT,
    turnover         DECIMAL(20,4),
    change_percent   DECIMAL(10,4),
    change_amount    DECIMAL(10,4),
    turnover_rate    DECIMAL(10,4),
    pe_ttm           DECIMAL(10,4),
    pb               DECIMAL(10,4),
    amplitude        DECIMAL(10,4),
    created_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_stock_daily UNIQUE (symbol, trade_date)
);
CREATE INDEX IF NOT EXISTS idx_stock_daily_symbol ON stock_daily_data(symbol);
CREATE INDEX IF NOT EXISTS idx_stock_daily_date   ON stock_daily_data(trade_date);

-- ============================================================
-- 3. 股票财务数据
-- ============================================================
CREATE TABLE IF NOT EXISTS stock_financial_data (
    id                      SERIAL PRIMARY KEY,
    symbol                  VARCHAR(10) NOT NULL,
    report_date             DATE        NOT NULL,
    report_type             VARCHAR(20),
    eps                     DECIMAL(10,4),
    eps_diluted             DECIMAL(10,4),
    total_revenue           DECIMAL(20,4),
    operating_profit        DECIMAL(20,4),
    net_profit              DECIMAL(20,4),
    total_assets            DECIMAL(20,4),
    total_liabilities       DECIMAL(20,4),
    equity                  DECIMAL(20,4),
    roe                     DECIMAL(10,4),
    gross_margin            DECIMAL(10,4),
    operating_cash_flow     DECIMAL(20,4),
    investing_cash_flow     DECIMAL(20,4),
    financing_cash_flow     DECIMAL(20,4),
    created_at              TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_stock_financial UNIQUE (symbol, report_date, report_type)
);
CREATE INDEX IF NOT EXISTS idx_stock_financial_symbol ON stock_financial_data(symbol);
CREATE INDEX IF NOT EXISTS idx_stock_financial_date   ON stock_financial_data(report_date);

-- ============================================================
-- 4. 股票技术指标（MA/MACD/KDJ/RSI/BOLL）
-- ============================================================
CREATE TABLE IF NOT EXISTS stock_indicators (
    id         SERIAL PRIMARY KEY,
    symbol     VARCHAR(10) NOT NULL,
    calc_date  DATE        NOT NULL,
    ma5        DECIMAL(12,4),
    ma10       DECIMAL(12,4),
    ma20       DECIMAL(12,4),
    ma60       DECIMAL(12,4),
    macd       DECIMAL(12,4),
    dif        DECIMAL(12,4),
    dea        DECIMAL(12,4),
    k          DECIMAL(12,4),
    d          DECIMAL(12,4),
    j          DECIMAL(12,4),
    rsi6       DECIMAL(12,4),
    rsi12      DECIMAL(12,4),
    rsi24      DECIMAL(12,4),
    boll_upper DECIMAL(12,4),
    boll_mid   DECIMAL(12,4),
    boll_lower DECIMAL(12,4),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- 预计算 lag 列（避免规则查询时跑窗口函数）
    ma5_lag1   DECIMAL(12,4),
    ma5_lag2   DECIMAL(12,4),
    ma5_lag3   DECIMAL(12,4),
    ma10_lag1  DECIMAL(12,4),
    ma10_lag2  DECIMAL(12,4),
    ma10_lag3  DECIMAL(12,4),
    ma20_lag1  DECIMAL(12,4),
    ma20_lag2  DECIMAL(12,4),
    ma20_lag3  DECIMAL(12,4),
    ma60_lag1  DECIMAL(12,4),
    k_lag1     DECIMAL(12,4),
    d_lag1     DECIMAL(12,4),
    dif_lag1   DECIMAL(12,4),
    dea_lag1   DECIMAL(12,4),

    -- 历史 OHLCV 的 lag（用于阳阴线、量比、净流入窗口判断）
    yang_lag0  BOOLEAN,
    yang_lag1  BOOLEAN,
    yang_lag2  BOOLEAN,
    yang_lag3  BOOLEAN,
    close_lag1 DECIMAL(12,4),
    close_lag2 DECIMAL(12,4),
    close_lag3 DECIMAL(12,4),
    close_lag5 DECIMAL(12,4),
    vol_lag1   DECIMAL(20,4),
    vol_lag2   DECIMAL(20,4),
    vol_lag3   DECIMAL(20,4),
    vol_lag5   DECIMAL(20,4),
    net_lag1   DECIMAL(20,4),
    net_lag2   DECIMAL(20,4),
    net_lag3   DECIMAL(20,4),

    -- 滚动统计（窗口聚合的预计算版本）
    vol_avg5    DECIMAL(20,4),
    low_min5    DECIMAL(12,4),
    low_min20   DECIMAL(12,4),
    low_min60   DECIMAL(12,4),
    high_max5   DECIMAL(12,4),
    high_max30  DECIMAL(12,4),
    high_max60  DECIMAL(12,4),
    high_max90  DECIMAL(12,4),

    CONSTRAINT uk_stock_indicators UNIQUE (symbol, calc_date)
);
CREATE INDEX IF NOT EXISTS idx_stock_indicators_symbol ON stock_indicators(symbol);
CREATE INDEX IF NOT EXISTS idx_stock_indicators_date   ON stock_indicators(calc_date);

-- ============================================================
-- 5. 股票资金流（按 symbol+date 唯一）
-- ============================================================
CREATE TABLE IF NOT EXISTS stock_money_flow (
    id                SERIAL PRIMARY KEY,
    symbol            VARCHAR(10) NOT NULL,
    trade_date        DATE        NOT NULL,
    main_net          DECIMAL(20,4),
    retail_net        DECIMAL(20,4),
    large_order_ratio DECIMAL(10,4),
    medium_order_ratio DECIMAL(10,4),
    small_order_ratio  DECIMAL(10,4),
    created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_stock_money_flow UNIQUE (symbol, trade_date)
);
CREATE INDEX IF NOT EXISTS idx_stock_money_flow_symbol ON stock_money_flow(symbol);
CREATE INDEX IF NOT EXISTS idx_stock_money_flow_date   ON stock_money_flow(trade_date);

-- ============================================================
-- 6. 股票资金流榜单（全市场排行、含 TimeSpan 维度）
-- ============================================================
CREATE TABLE IF NOT EXISTS stock_money_flow_all (
    id              SERIAL PRIMARY KEY,
    time_span       INTEGER     NOT NULL,            -- 0=即时 / 3 / 5 / 10
    serial_number   INTEGER,
    symbol          VARCHAR(10) NOT NULL,            -- 保留前导零
    name            VARCHAR(50),
    latest_price    DECIMAL(12,4),
    change_percent  DECIMAL(10,4),
    turnover_rate   DECIMAL(10,4),
    inflow_amount   DECIMAL(20,4),
    outflow_amount  DECIMAL(20,4),
    net_amount      DECIMAL(20,4),
    turnover        DECIMAL(20,4),
    trade_date      DATE        NOT NULL,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_stock_money_flow_all UNIQUE (symbol, trade_date, time_span)
);
CREATE INDEX IF NOT EXISTS idx_smf_all_symbol ON stock_money_flow_all(symbol);
CREATE INDEX IF NOT EXISTS idx_smf_all_date   ON stock_money_flow_all(trade_date);
CREATE INDEX IF NOT EXISTS idx_smf_all_span   ON stock_money_flow_all(time_span);

-- ============================================================
-- 7. 候选股表（规则执行结果）
-- ============================================================
CREATE TABLE IF NOT EXISTS target_trend_stock (
    id                SERIAL PRIMARY KEY,
    symbol            VARCHAR(10)  NOT NULL,
    name              VARCHAR(50),
    rule_name         VARCHAR(100) NOT NULL,
    rule_id           INTEGER,                       -- 可追溯 user_stock_rules.id
    user_id           UUID,                          -- 哪个用户定义的规则
    current_price     DECIMAL(12,4),
    change_3d         DECIMAL(10,4),
    change_percent    DECIMAL(10,4),
    turnover_rate     DECIMAL(10,4),
    net_inflow        DECIMAL(20,4),
    industry          VARCHAR(50),
    market            VARCHAR(20),
    matched_at        DATE        NOT NULL,
    created_at        TIMESTAMP   DEFAULT CURRENT_TIMESTAMP,

    -- 技术指标快照：规则命中时的 MA / MACD / KDJ / RSI / BOLL 当前值 + 昨日 lag
    ma5               DECIMAL(12,4),
    ma10              DECIMAL(12,4),
    ma20              DECIMAL(12,4),
    ma60              DECIMAL(12,4),
    ma5_prev          DECIMAL(12,4),
    ma10_prev         DECIMAL(12,4),
    ma20_prev         DECIMAL(12,4),
    macd              DECIMAL(12,4),
    dif               DECIMAL(12,4),
    dea               DECIMAL(12,4),
    k                 DECIMAL(12,4),
    d                 DECIMAL(12,4),
    j                 DECIMAL(12,4),
    rsi6              DECIMAL(12,4),
    rsi12             DECIMAL(12,4),
    rsi24             DECIMAL(12,4),
    boll_upper        DECIMAL(12,4),
    boll_mid          DECIMAL(12,4),
    boll_lower        DECIMAL(12,4),

    CONSTRAINT uq_target UNIQUE (symbol, rule_name, matched_at)
);

-- 兼容已存在的表：补齐新增技术指标列
ALTER TABLE target_trend_stock ADD COLUMN IF NOT EXISTS ma5        DECIMAL(12,4);
ALTER TABLE target_trend_stock ADD COLUMN IF NOT EXISTS ma10       DECIMAL(12,4);
ALTER TABLE target_trend_stock ADD COLUMN IF NOT EXISTS ma20       DECIMAL(12,4);
ALTER TABLE target_trend_stock ADD COLUMN IF NOT EXISTS ma60       DECIMAL(12,4);
ALTER TABLE target_trend_stock ADD COLUMN IF NOT EXISTS ma5_prev   DECIMAL(12,4);
ALTER TABLE target_trend_stock ADD COLUMN IF NOT EXISTS ma10_prev  DECIMAL(12,4);
ALTER TABLE target_trend_stock ADD COLUMN IF NOT EXISTS ma20_prev  DECIMAL(12,4);
ALTER TABLE target_trend_stock ADD COLUMN IF NOT EXISTS macd       DECIMAL(12,4);
ALTER TABLE target_trend_stock ADD COLUMN IF NOT EXISTS dif        DECIMAL(12,4);
ALTER TABLE target_trend_stock ADD COLUMN IF NOT EXISTS dea        DECIMAL(12,4);
ALTER TABLE target_trend_stock ADD COLUMN IF NOT EXISTS k          DECIMAL(12,4);
ALTER TABLE target_trend_stock ADD COLUMN IF NOT EXISTS d          DECIMAL(12,4);
ALTER TABLE target_trend_stock ADD COLUMN IF NOT EXISTS j          DECIMAL(12,4);
ALTER TABLE target_trend_stock ADD COLUMN IF NOT EXISTS rsi6       DECIMAL(12,4);
ALTER TABLE target_trend_stock ADD COLUMN IF NOT EXISTS rsi12      DECIMAL(12,4);
ALTER TABLE target_trend_stock ADD COLUMN IF NOT EXISTS rsi24      DECIMAL(12,4);
ALTER TABLE target_trend_stock ADD COLUMN IF NOT EXISTS boll_upper DECIMAL(12,4);
ALTER TABLE target_trend_stock ADD COLUMN IF NOT EXISTS boll_mid   DECIMAL(12,4);
ALTER TABLE target_trend_stock ADD COLUMN IF NOT EXISTS boll_lower DECIMAL(12,4);
ALTER TABLE target_trend_stock ADD COLUMN IF NOT EXISTS net_profit      DECIMAL(20,4);
ALTER TABLE target_trend_stock ADD COLUMN IF NOT EXISTS net_profit_yoy  DECIMAL(10,4);
ALTER TABLE target_trend_stock ADD COLUMN IF NOT EXISTS revenue_yoy     DECIMAL(10,4);
CREATE INDEX IF NOT EXISTS idx_target_symbol ON target_trend_stock(symbol);
CREATE INDEX IF NOT EXISTS idx_target_date   ON target_trend_stock(matched_at);
CREATE INDEX IF NOT EXISTS idx_target_rule   ON target_trend_stock(rule_name);

-- ============================================================
-- 8. 用户表
-- ============================================================
CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username      VARCHAR(50)  NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    email         VARCHAR(100) UNIQUE,
    phone         VARCHAR(20)  UNIQUE,
    is_active     BOOLEAN      DEFAULT TRUE,
    created_at    TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
);

-- 自动更新 updated_at
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_users_updated_at') THEN
        CREATE TRIGGER trg_users_updated_at BEFORE UPDATE ON users
        FOR EACH ROW EXECUTE FUNCTION set_updated_at();
    END IF;
END $$;

-- ============================================================
-- 9. 用户自选股
-- ============================================================
CREATE TABLE IF NOT EXISTS user_favorite_stocks (
    id          BIGSERIAL PRIMARY KEY,
    user_id     UUID         NOT NULL,
    symbol      VARCHAR(10)  NOT NULL,
    created_at  TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_user_fav UNIQUE (user_id, symbol)
);
CREATE INDEX IF NOT EXISTS idx_user_fav_symbol ON user_favorite_stocks(symbol);
CREATE INDEX IF NOT EXISTS idx_user_fav_user   ON user_favorite_stocks(user_id);

-- ============================================================
-- 10. 用户选股规则（JSONB 表达式）
-- ============================================================
CREATE TABLE IF NOT EXISTS user_stock_rules (
    id              SERIAL PRIMARY KEY,
    user_id         UUID         NOT NULL,
    rule_name       VARCHAR(100) NOT NULL,
    rule_expression JSONB        NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_user_rule UNIQUE (user_id, rule_name)
);
CREATE INDEX IF NOT EXISTS idx_user_rule_user     ON user_stock_rules(user_id);
CREATE INDEX IF NOT EXISTS idx_rule_expr_gin      ON user_stock_rules USING gin (rule_expression);

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_user_rules_updated_at') THEN
        CREATE TRIGGER trg_user_rules_updated_at BEFORE UPDATE ON user_stock_rules
        FOR EACH ROW EXECUTE FUNCTION set_updated_at();
    END IF;
END $$;

-- ============================================================
-- 11. 物化视图：stock_history_mv（日线 + 指标 + 资金流 三表对齐）
--     由 scripts/refresh_mv.py 负责创建/刷新
-- ============================================================

-- ============================================================
-- 兼容已部署表：补齐 stock_financial_data 同比增长率列
-- （唯一约束 uk_stock_financial (symbol, report_date, report_type) 已在建表语句里声明，
--   这里只补列，不重复加约束）
-- ============================================================
ALTER TABLE stock_financial_data ADD COLUMN IF NOT EXISTS net_profit_yoy DECIMAL(10,4);
ALTER TABLE stock_financial_data ADD COLUMN IF NOT EXISTS revenue_yoy    DECIMAL(10,4);


-- ============================================================
-- 12. 进阶精选 Top N 落库（每日精选结果留存，便于回看）
-- ============================================================
CREATE TABLE IF NOT EXISTS final_picks (
    id          BIGSERIAL PRIMARY KEY,
    trade_date  DATE        NOT NULL,
    rank        INT         NOT NULL,                 -- 1 / 2 / 3
    symbol      VARCHAR(10) NOT NULL,
    name        VARCHAR(50) NOT NULL,
    industry    VARCHAR(50),
    market      VARCHAR(20),
    score       INT         NOT NULL,
    breakdown   JSONB       NOT NULL,                 -- 6 维明细
    matched     TEXT        NOT NULL DEFAULT '[]',    -- 命中的预设 id
    created_at  TIMESTAMP   DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_final_picks UNIQUE (trade_date, symbol)
);
CREATE INDEX IF NOT EXISTS idx_final_picks_date ON final_picks(trade_date DESC);
CREATE INDEX IF NOT EXISTS idx_final_picks_symbol ON final_picks(symbol);
