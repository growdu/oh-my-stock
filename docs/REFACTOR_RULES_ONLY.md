# oh-my-stock 设计记录 · 规则 + 选股结果

> 本文档记录从「3 条线（热门/收藏/日线）+ 详情」重构为「1 条主线（规则筛股票）」的设计决策与最终实现。  
> 最近一次大规模重构在 `main` 分支 2026-08 批次（11 个 commit）。  
> 当前用户文档见 [USER_GUIDE.md](./USER_GUIDE.md)。

---

## 1. 最终目标

1. 整个产品只有一条主线：**用规则筛股票**。
2. 首次访问 `/` 就能看到 16 条系统预设的最新命中结果，用户可一键切换 / 执行。
3. 用户可以编辑、保存、自定义规则；执行结果即看即所得。

非目标（明确不做）：
- 实时行情推送（WebSocket）— 路线图中
- 复杂回测 / 组合优化
- 板块 / 行业轮动 / 因子分析
- 自选股管理（重构时已移除）

---

## 2. 最终页面

| 路径 | 组件 | 用途 |
|---|---|---|
| `/` | `pages/Display.vue` | 公开主页：单列堆叠规则 + 命中股票 + 技术指标条 + 行业筛选 + K 线弹窗 |
| `/` | `pages/Picker.vue` | **首页**：进阶精选 Top 3（评分 + 历史） |
| `/display` | `pages/Display.vue` | 全部预设 + 命中股票网格（公开） |
| `/admin/login` | `pages/LoginPage.vue` | 登录 |
| `/admin/rules` | `components/Rules.vue` | 自定义规则管理 + 可视化编辑器 |

已删除（与最初设计一致）：
- `HotStocks.vue` `Favorites.vue` `StockDailyTable.vue` `StockChart.vue` `StockPage.vue`

---

## 3. 系统预设（最终落地 16 条）

最初设计稿列了 7 条规则，落地为 **16 条** `backend/presets/presets.go` 里的 `All` 数组。这些预设：

- **不进用户表**（与最初设计稿不同）— 直接通过 `GET /api/v1/presets` 返回，命中通过 `POST /api/v1/presets/run` 执行
- 用户无需"系统恢复"按钮，因为它们在服务端常驻

按类别分组：

**量价类**：`consecutive-yang` `consecutive-yin` `main-inflow` `volume-amplify` `volume-shrink` `volume-ratio-blast` `turnover-active`
**估值类**：`quality-stocks`
**均线类**：`ma-golden-cross` `ma-death-cross` `ma-converge`
**指标类**：`oversold-bounce` `boll-bounce` `macd-histogram` 等
**形态类**：`volume-shrink-pullback` `limit-up-strong` `high-position-breakout`

详见 [README.md § 16 个内置预设](../README.md#16-个内置预设) 与 `backend/presets/presets.go`。

---

## 4. JSON DSL 演进

最初设计稿是 `{consecutive_yang_days: {gte: 3}, ...}` 平铺结构。落地版本改为**算子数组**结构，支持任意嵌套：

```jsonc
{
  "all":     [ /* AND 关系，全部满足 */ ],
  "any":     [ /* OR 关系，任一满足 (可选) */ ],
  "exclude": [ /* NOT 关系，全部排除 (可选) */ ]
}
```

每个算子形如 `{"type": "<name>", ...params}`。共 **28 个算子**，完整列表见 [README.md § 规则表达式语法](../README.md#规则表达式语法)。

### 4.1 与最初设计的差异

| 最初设计 | 最终实现 | 原因 |
|---|---|---|
| `consecutive_yang_days: {gte: 3}` | `{type: "yang_streak", days: 3}` | 算子统一 `type` 前缀，便于引擎 dispatch |
| `macd_golden_cross: {zone: "below_zero"}` | `{type: "macd_cross", location: "below_zero"}` | 同上，且复用 `macd_cross` 而非命名两个算子 |
| `exclude_st: true` | `{type: "is_st"}` 放在 `exclude` 数组 | 与其它排除条件合并表达 |
| `circ_mv_band: {min, max}` | `{type: "market_cap_yi", min, max}` | 单位明确为「亿」 |
| 黑名单"系统强制"前置 | `commonExcludes()` helper 写在每个 preset 的 `exclude` 数组 | 显式 > 隐式，方便用户编辑时看到 |

### 4.2 引擎架构（最终）

```
Run(db, expression, page, pageSize) →
  1. Compile(expression) → WHERE + args (递归编译 all/any/exclude)
  2. baseQuery = "WITH latest_dt AS (...),
                  ranked AS (SELECT ... FROM stock_history_mv h
                             LEFT JOIN stock_indicators i ON i.symbol=h.symbol AND i.calc_date=h.trade_date
                             LEFT JOIN LATERAL streak_* ...)
                  SELECT ... FROM ranked WHERE <compiled>"
  3. 命中 → upsert target_trend_stock（含 18 列技术指标快照）
  4. 返回 { rows, total, trade_date }
```

算子分类：

| 类别 | 算子 | 实现 |
|---|---|---|
| 简单比较 | `field`, `field_between` | 直接走 MV 列 |
| MA 系列 | `ma_compare`, `ma_alignment`, `ma_slope`, `ma_cross`, `close_vs_ma` | `*_lag1..3` 预计算列 |
| 量能 | `volume_ratio`, `volume_increasing`, `turnover_rate_range`, `change_percent_range`, `amount_range` | `vol_avg5`, `vol_lag*`, `turnover_rate` |
| K 线 | `yang_streak`, `yin_streak`, `breakout_high`, `low_breakout`, `close_position`, `cumulative_change` | `yang_lag*`, `high_max*`, `low_min*`, `close_lag*` |
| 指标 | `macd_cross`, `macd_histogram`, `kdj_cross`, `rsi_range`, `boll_position`, `boll_pct_b`, `boll_width`, `bias` | `dif/dea/k/d/j`, `rsi*`, `boll_*` + `*_lag1` |
| 涨跌停 | `limit_up`, `limit_down` | `change_percent` |
| 基础 | `board_in`, `industry_in/not_in`, `is_st/not_st`, `list_age_days_*`, `market_cap_yi`, `window_field` | 直接列或子查询 |

### 4.3 性能优化（重要）

最初设计稿担心窗口函数拖慢查询，落地时直接做了两件事：

1. **`scripts/compute_indicators.py` v5** —— 在 Python 侧预计算 `*_lag1..3` / `vol_avg5` / `high_max*` / `low_min*` / `yang_lag*` 等 25+ 列，存入 `stock_indicators`
2. **`backend/presets/runner.go` 去窗口** —— CTE 里 60+ 个 `LAG / OVER (ROWS BETWEEN ... PRECEDING)` 全部移除，改为 `LEFT JOIN stock_indicators` 读预计算列

效果：5000+ 股票全表跑一条规则从分钟级降到秒级。

---

## 5. 后端改造清单

### 5.1 文件

| 文件 | 动作 |
|---|---|
| `controllers/rule_runner_controller.go` | **重写**：命中时把 18 个技术指标快照写进 `target_trend_stock` |
| `controllers/stock_daily_data_controller.go` | 新增 `GET /:symbol/kline?days=N` 返回 OHLCV + MA + MACD + KDJ + RSI |
| `controllers/user_rule_controller.go` | 保留，提供自定义规则 CRUD |
| `models/target_trend_stock.go` | 新增 18 个技术指标快照字段 |
| `backend/presets/` | 新增整个目录（evaluator + runner + presets + test） |
| `scripts/create_table.sql` | 扩展 `stock_indicators` 25+ 预计算列；扩展 `target_trend_stock` 18 快照列；含 `ALTER ... IF NOT EXISTS` 兼容 |
| `scripts/compute_indicators.py` | v5：TRUNCATE + 重算所有 lag/rolling 列 |

### 5.2 不再做的事

- 不再 per-user 种入系统规则（系统预设在 `/api/v1/presets` 常驻）
- 不再有"系统规则 vs 自定义规则"的恢复按钮
- 不再有 `user_favorite_stocks` 表和 `/user/favorites*` 路由

---

## 6. 前端改造清单

### 6.1 `pages/Display.vue`（公开主页）

| 旧 | 新 |
|---|---|
| 8 条规则 carousel | 16 条规则**单列堆叠**（鼠标滚轮 / 触摸切换） |
| 命中股票只有价格 + 涨跌幅 | 命中卡片增加技术指标条 + 行业筛选 |
| 无 K 线 | 点击股票卡弹出 `KLineDialog`（蜡烛 + MA + 成交量 + MACD 三联图） |
| 无 MA 金叉死叉提示 | 客户端按 `ma*` / `ma*_prev` 字段判定并 chip 高亮 |

### 6.2 `components/Rules.vue`（管理后台）

- 保留 JSON 编辑能力（高级用户）
- 新增**可视化编辑器**（推荐）：按算子类型提供表单输入
- 「执行」按钮立即跑规则，结果展示在下方卡片网格（带技术指标条）

### 6.3 `components/KLineDialog.vue`（新增）

ECharts 蜡烛图 + MA + 成交量（含 5 日均量）+ MACD 三联图。
- 顶部 toolbar：股票基础信息 + 现价 + 涨跌幅 + 30/60/90/180 日切换
- tooltip 同时显示 OHLCV / MA / 五日均价 / DIF / DEA / MACD / KDJ / RSI
- 配色统一阳线红 / 阴线绿

---

## 7. 数据初始化

最初设计稿依赖 `docker-compose seed-data` 任务。落地后改为：

```bash
# 一次性初始化
cd scripts && ./daily_refresh.sh --initial

# 每天定时
IMMEDIATE_RUN=0 python timer.py
```

`daily_refresh.sh` 内部串联：
1. `get_basic_info_lite.py`（仅 `--initial`）
2. `fetch_daily.py` 新浪 K 线（替代 akshare，更快）
3. `fetch_money_flow.py` 东财资金流
4. `compute_indicators.py` v5 全量重算指标
5. `refresh_mv.py` 增量刷新 MV

旧脚本 `get_stock_daily.py` / `get_money_flow_v2.py` / `get_financial_info.py` 保留在仓库作为兼容入口，不在 README 主流程出现。

---

## 8. 验收清单（已通过）

- [x] 首次访问 `/` 看到 16 条系统预设
- [x] 鼠标滚轮 / 触摸可切换规则
- [x] 命中卡片显示 MA / KDJ / RSI / BOLL + 金叉死叉标
- [x] 行业筛选 chip 可过滤命中结果
- [x] 点击股票卡弹出 K 线（三联图）
- [x] `/admin/rules` 可视化编辑器可生成 JSON 并保存
- [x] 自定义规则执行后下方展示命中 + 技术指标条
- [x] 5000+ 股票跑一条规则 < 2 秒（基于预计算列）

---

## 9. 后续路线

| 优先级 | 项目 | 状态 |
|---|---|---|
| P0 | `daily_refresh.sh` 一键日刷 | ✅ 已完成 |
| P0 | .gitignore 补构建产物 / 本地配置 | ✅ 已完成 |
| P1 | 自定义规则可视化编辑器 | ✅ 已完成 |
| P1 | K 线图三联图（蜡烛 + 量 + MACD） | ✅ 已完成 |
| P1 | 文档同步（README / USER_GUIDE） | ✅ 已完成 |
| P2 | 规则回测 API（对历史 N 天每天跑一次规则） | ⏳ 待办 |
| P2 | WebSocket / 轮询推送新命中 | ⏳ 待办 |
| P2 | 复权处理（qfq 因子） | ⏳ 待办 |
| P3 | 板块 / 行业聚合视图 | ⏳ 待办 |
| P3 | 单元测试覆盖 controllers 层 | ⏳ 待办 |
