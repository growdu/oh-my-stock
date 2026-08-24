# oh-my-stock

个人股票分析系统。  
采集 → 入库 → 计算指标 → 刷新物化视图 → 规则筛选 → Web 可视化。

主页一条主线：**用规则筛股票**。16 条内置预设 + 自定义规则，所有命中结果直接展示 MA / KDJ / RSI / BOLL 等技术形态，点击可看 K 线。

## 技术栈

| 层 | 技术 |
|---|---|
| 前端 | Vue 3 + Vite + Element Plus + ECharts |
| 后端 | Go 1.24 + Gin + GORM + PostgreSQL |
| 数据采集 | Python 3.12 + requests + pandas + SQLAlchemy |
| 鉴权 | 自实现 HMAC-SHA256 JWT（零外部依赖） |
| 部署 | Docker / docker compose |

## 页面

| 路径 | 用途 | 鉴权 |
|---|---|---|
| `/` | 公开展示页：16 个预设单列堆叠 + 命中股票网格（带技术指标 + 行业筛选 + K线弹窗） | 公开 |
| `/admin/login` | 管理后台登录 | 公开 |
| `/admin/results` | 管理后台的展示页（与 `/` 视图基本一致） | JWT |
| `/admin/rules` | 自定义规则管理：可视化编辑 + 立即执行 | JWT |

## 快速开始（Docker）

```bash
cp .env.example .env                 # 改密码 / JWT secret
docker compose up -d                 # 一键启动 pg + 后端 + 前端 + 物化视图
open http://localhost:5173
```

## 快速开始（手动开发）

### 1) PostgreSQL

```bash
psql "$DATABASE_URL" -f scripts/create_table.sql
psql "$DATABASE_URL" -f scripts/refresh_mv.sql
```

### 2) 后端

```bash
cd backend
cp .env.example .env                  # 填数据库 & JWT secret
export $(cat .env | xargs)
go run .                              # 监听 :3003，Swagger: http://localhost:3003/swagger/index.html
```

### 3) 前端

```bash
cd front
yarn install
cp .env.example .env                  # 可选，覆盖 VITE_API_BASE
yarn dev                              # 默认代理到 192.168.3.99:3003
yarn build                            # 生产构建到 dist/
```

### 4) 数据采集

```bash
cd scripts
python -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt

# 一键全量初始化（首次部署）
./daily_refresh.sh --initial

# 每天定时调度（17:00 由 systemd 接管；machine restart 自动续上）
bash deploy/systemd/install.sh   # 一键装两个 .timer
sudo systemctl list-timers oh-my-stock-*   # 看下一次触发
```

> **数据源提示**：当前实现默认走 Tencent `web.ifzq.gtimg.cn` (主) + Eastmoney 备用 + Sina 兜底，Sina 在部分 IP 被限频时可由 `SOURCES` 常量调整优先级。
> **网络限制**：`fetch_money_flow.py` 依赖 `push2his.eastmoney.com`，部分出口 IP 会被该接口掐断。脚本会先做一次 5 秒探针，命中则跳过整步（~2 秒退出），不阻塞 K 线 / 指标 / MV；`main-inflow` 预设可能因数据稀疏返回空结果。

`daily_refresh.sh` 内部依次执行：
1. `get_basic_info_lite.py` —— 仅 `--initial` 时跑，拉沪深京全量代码+名称
2. `fetch_daily.py` —— 新浪 K 线（多线程 5000+ 只 / 60 日）
3. `fetch_money_flow.py` —— 东财主力资金流（5 日）
4. `compute_indicators.py` —— 全量重算 MA/MACD/KDJ/RSI/BOLL + 预计算 lag/rolling 列
5. `refresh_mv.py` —— 增量刷新 `stock_history_mv`

支持 `--skip-fetch` / `--skip-compute` / `--skip-mv` 跳过单步。

## 目录结构

```
.
├── backend/                 Go HTTP API
│   ├── config/              配置 + HMAC JWT
│   ├── controllers/         Gin 控制器层（含 stock_daily_data / rule_runner / presets）
│   ├── models/              GORM 数据模型（含 target_trend_stock 指标快照）
│   ├── presets/             规则引擎（evaluator / runner / presets 16 条）
│   ├── middleware/          JWT 中间件
│   ├── docs/                swag 生成的 OpenAPI
│   └── main.go
├── front/                   Vue 3 SPA
│   ├── src/
│   │   ├── pages/           Display.vue / Results.vue / LoginPage.vue / StockPage.vue
│   │   ├── components/      Rules.vue / KLineDialog.vue
│   │   ├── composables/     usePresetCache / useCard3D
│   │   ├── utils/api/       axios 封装
│   │   └── router/
│   └── nginx.conf
├── scripts/
│   ├── create_table.sql     全量 DDL（10 张表 + 物化视图）
│   ├── refresh_mv.sql       stock_history_mv DDL
│   ├── refresh_mv.py        物化视图创建/刷新
│   ├── get_basic_info_lite.py   精简版基础信息（按前缀填 market）
│   ├── fetch_daily.py       新浪 K 线 fetcher
│   ├── fetch_money_flow.py  东财资金流 fetcher
│   ├── fetch_quote.py       实时行情 fetcher
│   ├── compute_indicators.py  v5：单股全量指标 + 预计算 lag/rolling 列
│   ├── daily_refresh.sh     一键串联数据管道（首选入口）
│   ├── timer.py             调度器（保留作为应急，systemd 接管后不依赖）
├── deploy/
│   └── systemd/             systemd .timer / .service (Mon-Fri 17:00 & 18:00)
├── scripts/cache/           沪深京股票清单 CSV 缓存
├── docs/
│   ├── 库表设计.md
│   ├── REFACTOR_RULES_ONLY.md
│   └── USER_GUIDE.md        用户使用指南
├── deploy/                  docker-compose 一键部署脚本
├── docker-compose.yml
└── README.md
```

## 接口列表（节选）

| Method | Path | 说明 | 鉴权 |
|---|---|---|---|
| POST | /api/v1/user/register | 注册 | 公开 |
| POST | /api/v1/user/login    | 登录 → 返 token | 公开 |
| GET  | /api/v1/user/rules          | 自定义规则列表 | JWT |
| POST | /api/v1/user/rules          | 新增规则 | JWT |
| PUT  | /api/v1/user/rules/:id      | 修改 | JWT |
| DELETE | /api/v1/user/rules/:id    | 删除 | JWT |
| POST | /api/v1/user/rules/preview  | 预览规则（不入库） | JWT |
| POST | /api/v1/user/rules/:id/run  | 执行规则 → 写入 target_trend_stock | JWT |
| GET  | /api/v1/presets              | 16 个系统预设 | 公开 |
| POST | /api/v1/presets/run          | 执行预设，返回命中列表 | 公开 |
| GET  | /api/v1/stocks/list          | 股票列表（分页） | 公开 |
| GET  | /api/v1/stocks/search?q=     | 模糊搜索 | 公开 |
| GET  | /api/v1/stocks/hot           | 热门（涨幅≥5%） | 公开 |
| GET  | /api/v1/stock-daily-data/:symbol/kline?days=N | 单股 K 线 + MA/MACD/KDJ/RSI | 公开 |
| GET  | /api/v1/target-stocks?rule_name=&page= | 候选股 | 公开 |

完整 OpenAPI 见 `http://localhost:3003/swagger/index.html`

## 16 个内置预设

| ID | 名称 | 思路 |
|---|---|---|
| `consecutive-yang` | 连续阳线 | 连续 N 天收阳 |
| `consecutive-yin` | 连续阴线 | 连续 N 天收阴 |
| `main-inflow` | 主力连续流入 | 连续 N 天主力净流入 |
| `volume-amplify` | 连续放量 | 连续 N 天量比放大 |
| `volume-shrink` | 连续缩量 | 连续 N 天量比缩小 |
| `volume-ratio-blast` | 单日放量 | 量比 >= 1.5 |
| `turnover-active` | 高换手活跃 | 换手 3~15% |
| `quality-stocks` | 稳健基本面 | PE 0~80 & PB < 10 |
| `ma-golden-cross` | 均线金叉 | MA5 上穿 MA10 + 站上 MA20 + MA20 上扬 |
| `ma-death-cross` | 均线死叉 | MA5 下穿 MA20 + 跌破 MA60 |
| `oversold-bounce` | 超卖反弹 | RSI6<30 + KDJ 金叉 + 站上 MA5 |
| `ma-converge` | 均线粘合 | MA5/10/20 乖离<2% + 换手 1~15% |
| `boll-bounce` | BOLL 中轨反弹 | BOLL 下轨 + 站上 MA5 + MACD 柱转正 |
| `volume-shrink-pullback` | 缩量回踩 MA20 | 量比<=0.8 + 换手<5% |
| `limit-up-strong` | 强势涨停 | >=9.8% + 站上 MA20 + 量比>=1.5 |
| `high-position-breakout` | 高位突破 | 60 日高位 80~100% + MA 多头 + 放量 |

所有预设的完整 JSON 见 `backend/presets/presets.go`。

## 规则表达式语法

顶层结构：

```jsonc
{
  "all":     [ /* AND 关系，全部满足 */ ],
  "any":     [ /* OR 关系，任一满足 */ ],
  "exclude": [ /* NOT 关系，全部排除 */ ]
}
```

支持的算子（28 个）：

| 算子 | 字段 / 参数 | 说明 |
|---|---|---|
| `field` | `name`, `op`, `value` | 简单字段比较 |
| `field_between` | `name`, `min`, `max` | 区间 |
| `ma_compare` | `fast`, `slow`, `op` | MA 之间的 > / < |
| `ma_alignment` | `order[]` | MA 多头/空头排列 |
| `ma_slope` | `ma`, `days`, `op` | MA 在 N 日内上扬/下倾 |
| `ma_cross` | `fast`, `slow`, `direction` | 均线金叉/死叉 |
| `close_vs_ma` | `ma`, `op` | 收盘价相对某 MA |
| `volume_ratio` | `min`, `max` | 量比 (latest.volume / vol_avg5) |
| `volume_increasing` | `days`, `min_ratio` | 连续放量 |
| `turnover_rate_range` | `min`, `max` | 换手率区间 |
| `change_percent_range` | `min`, `max` | 涨跌幅区间 |
| `amount_range` | `min`, `max` | 成交额区间 |
| `yang_streak` / `yin_streak` | `days` | 连续阳/阴线 |
| `cumulative_change` | `days`, `max_pct` | N 日累计涨跌幅 |
| `breakout_high` | `lookback` | 突破 N 日新高 |
| `low_breakout` | `lookback` | 跌破 N 日新低 |
| `close_position` | `lookback`, `min`, `max` | 收盘在 N 日区间位置 |
| `macd_cross` | `location` | MACD 金叉/死叉 |
| `macd_histogram` | `sign`, `growing` | MACD 柱符号 + 放大 |
| `kdj_cross` | `location` | KDJ 金叉/死叉 |
| `rsi_range` | `min`, `max` | RSI 区间 |
| `boll_position` | `position` | BOLL 上/中/下轨附近 |
| `boll_pct_b` | `min`, `max` | %b 位置 |
| `boll_width` | `min`, `max` | 通道宽度 |
| `bias` | `ma`, `min`, `max` | 乖离率 |
| `limit_up` / `limit_down` | `min_pct` / `max_pct` | 涨停 / 跌停 |
| `board_in` | `boards[]` | 板型筛选（主板/创业板/科创板/北交所） |
| `industry_in` / `industry_not_in` | `industries[]` | 行业 IN / NOT IN |
| `is_st` / `is_not_st` | — | 排除 ST |
| `list_age_days_gte` / `_lt` | `days` | 上市天数 |
| `market_cap_yi` | `min`, `max` | 总市值（亿元） |
| `window_field` | `name`, `op`, `value`, `lookback` | N 日窗口内的字段比较 |

## 用户文档

- [USER_GUIDE.md](docs/USER_GUIDE.md) —— 使用指南（基于实际页面截图式描述）
- [REFACTOR_RULES_ONLY.md](docs/REFACTOR_RULES_ONLY.md) —— 规则 + 选股结果重构设计
- [库表设计.md](docs/库表设计.md) —— 数据库表结构

## 路线图

- [x] 全量 DDL（10 张表 + 物化视图 + 25+ 预计算指标列）
- [x] Go HTTP API + Swagger
- [x] HMAC 自实现 JWT
- [x] 16 条系统预设
- [x] 28+ 算子的规则引擎
- [x] 技术指标自动计算（含 lag/rolling 预计算）
- [x] 物化视图自动刷新
- [x] Docker 一键启动
- [x] 前后端打通（K 线 + 自定义规则 + 预设 + 候选 + 技术指标可视化）
- [x] 自定义规则可视化编辑器
- [x] 单股 K 线图（蜡烛 + MA + 成交量 + MACD 三联图）
- [x] 一键日刷脚本 `daily_refresh.sh`
- [ ] 单元测试覆盖 controllers 层
- [ ] 实时推送（WebSocket）
- [ ] 规则回测
- [ ] 复权处理
