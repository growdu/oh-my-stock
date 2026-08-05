# oh-my-stock 重构设计 · 规则 + 选股结果

> 范围：去掉「热门股票 / 收藏 / 日线」三条线，整个界面只剩 **规则编辑** 与 **该规则筛出的股票** 两个核心。

---

## 1. 目标

1. 整个产品只有一条主线：**用规则筛股票**。
2. 首次登录默认就能看到内置 7 条规则的最新命中结果，用户可一键执行。
3. 用户可以编辑、保存、自定义规则；执行结果即看即所得。

非目标（本期不做）：
- 实时行情推送
- K 线 / 技术指标图表（用户已确认无需）
- 自选股管理（用户已确认无需）
- 复杂回测、组合优化

---

## 2. 页面与路由

### 2.1 删除

| 旧路径 | 删除组件 | 说明 |
| --- | --- | --- |
| `/hot` | `front/src/components/HotStocks.vue` | 涨幅榜 / 规则候选切换 |
| `/favorites` | `front/src/components/Favorites.vue` | 自选股 |
| `/daily` | `front/src/components/StockDailyTable.vue` | 日线查询 |
| `/stock` | `front/src/pages/StockPage.vue` + `components/StockChart.vue` | 综合详情 |
| `/user/favorites*` | `controllers/user_favorite_controller.go` + 表 `user_favorite_stocks` | 后端一并清理 |

### 2.2 保留 / 重写

| 路径 | 组件 | 说明 |
| --- | --- | --- |
| `/login` | `LoginPage.vue` | 不动 |
| `/home` | `HomePage.vue`（**重写**） | 默认页：选股结果（按所选规则） |
| `/rules` | `Rules.vue`（**重写**） | 规则 CRUD + 系统预设 |
| Header 菜单 | `Header.vue` | 仅保留 首页 / 规则 |

### 2.3 新增

| 路径 | 组件 | 说明 |
| --- | --- | --- |
| `/rules/:id` | `RuleDetailPage.vue` | 单条规则详情 + 命中股票表（默认即 `/home` 选中规则时切换） |

---

## 3. 默认规则（系统预设）

按用户提供的内容，落地为 7 条内置规则，**每个用户登录时自动种入其 `user_stock_rules` 表**，并标记 `is_system=TRUE`，用户可在「规则」页一键重置/查看 JSON。

| # | 名称 | 章节 | 关键条件 |
| --- | --- | --- | --- |
| 1 | 长期下跌后底部反转 | 二 | 连续 3 日主力净流入 + 3 阳 + 3 日放量 + 3 日累计涨幅 ≤15% + 站上 MA5 + MACD 零下金叉 + 换手 3–8% |
| 2 | 突破三个月高位 | 三 | close > max(high,60d) + 量比 ≥1.5 + 当日涨幅 3–7% + 站上 MA20 且 MA20 上行 |
| 3 | 均线多头排列 | 四 | MA5>MA10>MA20>MA60 + 四条均线 5 日斜率均向上 |
| 4 | 量价齐升 | 五 | 量比 >1.5 + 换手 3–8% + 近 5 日均价逐日抬高 |
| 5 | 技术指标共振 | 八 | KDJ J 上穿 K、D 且 J<50 + MACD 零上金叉 + RSI 在 40–70 |
| 6 | 强势龙头股 | 七 | 流通市值 50–300 亿 + 当日涨幅 ≥5% + 换手 ≥5%（近似，缺龙虎榜） |
| 7 | 黑名单（通用排除） | 九 | 上市 <60 日 + ST/*ST + 流通市值 <20 亿或 >1000 亿（联合在其他规则前过滤） |

> 第六条「基本面筛选」本期**不做**：财务表缺 PE-TTM/资产负债率/商誉字段，且 AKShare 财报同步是另一条独立流水线，先保留接口位置。

---

## 4. JSON DSL 扩展

在现有 `market / industry / symbol_prefix / change_percent / turnover_rate / current_price / consecutive_up_days / consecutive_inflow_days / consecutive_volume_amplify_days` 之外，新增以下算子。

### 4.1 新算子（命名沿用 snake_case）

| key | 含义 | 取值 | 备注 |
| --- | --- | --- | --- |
| `consecutive_yang_days` | 连续阳线天数 | `{gte, lte}` 整数 | close > open |
| `total_3day_change_le` | 3 日累计涨幅上限 | 数字（百分比） | (C(t)-C(t-3))/C(t-3) |
| `close_above_ma5` | 收盘价站上 MA5 | `true` | 布尔 |
| `close_above_ma20` | 同上 MA20 | `true` | |
| `macd_golden_cross` | MACD 金叉 | `{zone: "below_zero" \| "above_zero" \| "any"}` | 今日 DIF>DEA 且昨日 DIF≤DEA |
| `kdj_low_golden_cross` | KDJ 低位金叉 | `{j_max: 50}` | 今日 J>K 且昨日 J≤K，且今日 J<阈值 |
| `rsi_band` | RSI 区间 | `{min: 40, max: 70}` | 取 RSI6 |
| `break_60day_high` | 突破近 60 日新高 | `true` | close > max(high, t-1..t-60) |
| `volume_ratio` | 量比 | `{gte, lte}` | V(t) / AVG(V, t-5..t-1) |
| `day_change_range` | 当日涨幅区间 | `{min: 3, max: 7}` | 单位 % |
| `ma_alignment_bull` | 均线多头排列 | `true` | MA5>MA10>MA20>MA60 |
| `ma_slope_up` | 各 MA 5 日斜率向上 | `true` | MA(n, t) > MA(n, t-5) for n∈{5,10,20,60} |
| `days_listed_gte` | 上市天数下限 | 整数 | 当前日期 - listing_date ≥ N |
| `exclude_st` | 排除 ST/*ST | `true` | name LIKE 'ST%' 或 '*ST%' |
| `exclude_new_stock` | 排除次新股 | `true` | 同 `days_listed_gte: 60` |
| `circ_mv_band` | 流通市值区间（亿） | `{min, max}` | 需要 basic_info.outstanding_shares * 当前价 |

### 4.2 黑名单统一前置

引擎在每条规则执行前，自动追加：

```json
{
  "exclude_st": true,
  "exclude_new_stock": true,
  "circ_mv_band": {"min": 20, "max": 1000}
}
```

无开关（系统强制）。

### 4.3 示例（底部反转规则 JSON）

```json
{
  "exclude_st": true,
  "exclude_new_stock": true,
  "circ_mv_band": {"min": 20, "max": 1000},
  "consecutive_inflow_days": {"gte": 3},
  "consecutive_yang_days":   {"gte": 3},
  "consecutive_volume_amplify_days": {"gte": 3, "min_ratio": 1.2},
  "total_3day_change_le": 15,
  "close_above_ma5": true,
  "macd_golden_cross": {"zone": "below_zero"},
  "turnover_rate": {"between": [3, 8]}
}
```

---

## 5. 后端改造

### 5.1 文件级动作

| 文件 | 动作 |
| --- | --- |
| `controllers/rule_runner_controller.go` | **重写**：把 SQL 拼装改为「主 select + JSON DSL → WHERE + HAVING」；新增 4.1 全部算子 |
| `controllers/user_rule_controller.go` | 增 `SeedSystemRules(uid)`、`ResetSystemRules(uid)`；新建/更新时不再覆盖系统规则 |
| `controllers/user_favorite_controller.go` | **删除** |
| `controllers/stock_history_controller.go` | 保留 `/stocks/list`，但默认按「selected rule」+分页 |
| `controllers/stock_basic_info_controller.go` | 删除或保留 GET 单只，仅 admin 用 |
| `middleware/...` | 不动 |
| `models/...` | `UserStockRule` 加 `is_system BOOLEAN DEFAULT FALSE`、`description TEXT`、`template VARCHAR(50)` |
| `scripts/refresh_mv.sql` | **扩展 MV**：新增 `prev_close`、`max_high_60d`、`ma5_prev5`、`ma10_prev5`、`ma20_prev5`、`ma60_prev5`、`avg_volume_5d`、`yesterday_dif`、`yesterday_dea`、`yesterday_k`、`yesterday_d`、`yesterday_j`、`circ_mv` 等列；让 4.1 算子尽量走 MV，避免运行期窗口函数 |
| `scripts/create_table.sql` | `user_stock_rules` 加字段；`user_favorite_stocks` 表删除（无引用即可） |
| `main.go` | 移除 `/user/favorites*` 路由组 |
| `seed.go` | 启动时不再 seed favorite |
| `target_trend_stock` 表 | 维持，新增 `matched_reason JSONB` 字段，存每条命中是哪些算子通过的（调试用） |

### 5.2 规则引擎架构（重写后）

```
runRule(rule) →
  1. resolvePreBlacklist(spec) → spec  // 系统黑名单合并
  2. baseQuery = "SELECT symbol, ... FROM stock_history_mv_latest WHERE trade_date = latest"
  3. buildWhere(spec) → WHERE sql + args
  4. 命中 → upsert target_trend_stock(matched_at=today, matched_reason=spec)
  5. 返回 { matched: N, date, rules: [...] }
```

算子分类：

- **简单比较**：`change_percent`、`turnover_rate`、`current_price`、`pe_ttm` 等直接走 MV 列
- **窗口聚合**：`consecutive_*_days`、`break_60day_high`、`ma_slope_up` 等通过 CTE 或 MV 预计算列
- **布尔**：`close_above_ma5`、`ma_alignment_bull`、`exclude_st` 直接走 MV 列或简单表达式

### 5.3 系统预设种入

- 用户首次登录成功后，后端 `Login` 末尾检查 `SELECT COUNT(*) FROM user_stock_rules WHERE user_id=? AND is_system=TRUE`；若为 0，调用 `SeedSystemRules(uid)`。
- `SeedSystemRules` 写入 7 条规则，命名带 `[系统]` 前缀；`template='bottom_reverse' / 'break_60d_high' / ...`。
- 用户在前端删除系统规则不会真删，而是设 `is_active=FALSE`；提供「恢复系统规则」按钮。

---

## 6. 前端改造

### 6.1 路由

```
/login                LoginPage
/                     redirect → /home
/home                 HomePage（按当前选中规则展示结果）
/rules                RulesPage（系统 + 自定义规则 CRUD）
/rules/:id            RuleDetailPage（单条规则的命中明细；可由 /rules 跳转）
```

### 6.2 组件

| 组件 | 状态 |
| --- | --- |
| `Header.vue` | 简化菜单：首页 / 规则 |
| `HomePage.vue` | **重写**：顶部规则下拉（来自 /user/rules）+ 执行按钮 + 命中股票表（symbol、name、change%、turnover%、net_inflow、industry、market） |
| `Rules.vue` | **重写**：左列「系统预设」只读卡片 + 右列「我的规则」可编辑 JSON；底部「+ 新建规则」按钮；模板选择器（7 种） |
| `LoginPage.vue` | 不动 |
| `HotStocks.vue / Favorites.vue / StockChart.vue / StockDailyTable.vue / StockPage.vue` | **删除** |

### 6.3 交互细节

- HomePage 顶部一个 `el-select`（选规则）+ `执行` 按钮 + 「最新执行于 HH:mm」展示
- 表格分页（沿用现成的 `usePaginatedData`，适配 `target-stocks` 接口）
- Rules.vue 编辑 JSON 用 `el-input type=textarea`，右侧提供「从模板生成」按钮（点选模板后自动填入默认 JSON）
- 所有按钮 loading/disabled 状态完整，错误用 `ElMessage`

---

## 7. 数据初始化与一键体验

为支持"装上就能跑"，新增 deploy 子任务：

### 7.1 `docker-compose.yml` 新增 `seed-data` 一次性任务

```yaml
seed-data:
  image: python:3.11-slim
  env_file: [.env]
  volumes:
    - ./scripts:/scripts:ro
    - ./logs:/logs
  working_dir: /scripts
  depends_on:
    postgres:
      condition: service_healthy
  entrypoint: ["bash", "-lc"]
  command: |
    pip install -q -r requirements.txt &&
    python get_basic_info.py &&
    python get_stock_daily.py &&
    python get_money_flow_v2.py &&
    python compute_indicators.py &&
    python refresh_mv.py
  restart: "no"
```

### 7.2 同步脚本端到端链路

1. `get_basic_info.py` — 全 A 股基础信息（约 5400 条）→ `stock_basic_info`
2. `get_stock_daily.py` — 最近 60 个交易日日线 → `stock_daily_data`
3. `get_money_flow_v2.py` — 资金流 → `stock_money_flow` & `stock_money_flow_all`
4. `compute_indicators.py` — MA/MACD/KDJ/RSI/BOLL → `stock_indicators`
5. `refresh_mv.py` — 重建 MV（含新增列）

`deploy.sh start` 流程改为：建表 → seed-data → 启 backend → 物化视图首次 refresh。

> 注意：首次抓全 A 股日线 + 资金流脚本可能要 30–60 分钟；首次启动会慢，之后命中 `pgdata` 卷很快。

---

## 8. 验收清单

- [ ] 登录默认进入 `/home`，能看到 7 条系统规则的列表（默认选中第一条）
- [ ] 点「执行」，表格 1–2 秒内出结果
- [ ] `/rules` 能看到 7 条「[系统] xxx」+ 0 条自定义
- [ ] 新建一条自定义规则（基于「均线多头」模板），保存后回到 `/home` 选中该规则执行，能出结果
- [ ] 删除某条系统规则后，「恢复系统规则」按钮可恢复
- [ ] 后端日志无 ERROR，规则引擎每个新算子有 ≥1 个测试 SQL（手工跑通）

---

## 9. 实施分阶段

| 阶段 | 内容 | 估计 |
| --- | --- | --- |
| P1 后端：模型 + 路由清理 | `UserStockRule` 加字段；删 favorites；路由清理；MV 扩展 | 半天 |
| P2 后端：规则引擎重写 | 4.1 全部算子实现 + 单元测试（手工 SQL 验证） | 1 天 |
| P3 后端：系统预设种入 | SeedSystemRules、首次登录触发、Reset API | 半天 |
| P4 前端：路由 + 删组件 | 删 5 个组件/页面，Header 简化 | 1 小时 |
| P5 前端：HomePage 重写 | 规则下拉 + 执行 + 结果表 | 半天 |
| P6 前端：RulesPage 重写 | 模板选择器 + JSON 编辑 + 系统/自定义分区 | 1 天 |
| P7 数据：seed-data 任务 + 首次同步 | docker-compose、deploy.sh、跑通全链路 | 半天（不计算抓数据时长） |

合计 ≈ 4 个工作日。

---

## 10. 风险与待确认

1. **首次数据同步耗时**：全 A 股日线 + 资金流首次抓可能 30–60 分钟，需要确认沙箱/服务器网络可达 AKShare。
2. **AKShare 接口稳定性**：脚本已有 try/except + 日志，但若整日拉取失败，需要给出"近一日数据"提示。
3. **流通市值字段**：`outstanding_shares` 是 `DECIMAL(20,4)`（万股？股？需核对脚本），需在 MV 里用 `outstanding_shares * close / 1e8` 折算成"亿"。
4. **MACD/KDJ 金叉判定**：用昨日 vs 今日的 dif/dea、k/d/j 比较；需要 MV 增加 `yesterday_*` 字段或运行期用窗口。
5. **强势龙头**：龙虎榜本期不做近似（流通市值 + 涨幅 + 换手）；用户接受后写进规则。
