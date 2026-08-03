# oh-my-stock

个人股票分析系统。  
采集 → 入库 → 计算指标 → 刷新物化视图 → 规则筛选 → Web 可视化。

## 技术栈

| 层 | 技术 |
|---|---|
| 前端 | Vue 3 + Vite + Element Plus + ECharts |
| 后端 | Go 1.24 + Gin + GORM + PostgreSQL |
| 数据采集 | Python 3.12 + AKShare + pandas + SQLAlchemy |
| 鉴权 | 自实现 HMAC-SHA256 JWT（零外部依赖） |
| 部署 | Docker / docker compose |

## 快速开始（Docker）

```bash
cp .env.example .env                 # 改密码 / JWT secret
docker compose up -d                 # 一键启动 pg + 后端 + 前端 + 物化视图
# 等 init-mv 完成后
docker compose run --rm -T backend /app/oh-my-stock &  # 可选：启动后端服务
# 浏览器打开
open http://localhost:5173
```

## 快速开始（手动开发）

### 1) PostgreSQL

```bash
# 1) 启动一个 PG 实例（任何方式都行），准备好 DATABASE_URL
psql "$DATABASE_URL" -f scripts/create_table.sql
psql "$DATABASE_URL" -f scripts/refresh_mv.sql
```

### 2) 后端

```bash
cd backend
cp .env.example .env                  # 填数据库 & JWT secret
export $(cat .env | xargs)            # 注入到当前 shell
go run .
# 浏览器打开 http://localhost:3003/swagger/index.html
```

### 3) 前端

```bash
cd front
yarn install
cp .env.example .env  # 可选，覆盖 VITE_API_BASE
yarn dev              # 开发模式 (默认代理到 192.168.3.99:3003)
yarn build            # 生产构建到 dist/
```

### 4) 数据采集（一次性或定时）

```bash
cd scripts
python -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt

# 一次性初始化（顺序）
python get_basic_info.py            # 1) 拉全部 A 股基础信息
python get_stock_daily.py            # 2) 拉最新 3 个交易日日线
python get_money_flow_v2.py          # 3) 拉资金流榜单
python get_financial_info.py         # 4) 拉财报
python compute_indicators.py         # 5) 计算 MA/MACD/KDJ/RSI/BOLL
python refresh_mv.py                 # 6) 创建/刷新物化视图

# 每天定时调度（16:00 起）
IMMEDIATE_RUN=0 python timer.py
```

## 目录结构

```
.
├── backend/                 Go HTTP API
│   ├── config/              配置 + HMAC JWT
│   ├── controllers/         Gin 控制器层
│   ├── models/              GORM 数据模型
│   ├── middleware/          JWT 中间件
│   ├── docs/                swag 生成的 OpenAPI 文档
│   ├── main.go
│   ├── go.mod
│   ├── config.json          配置（支持 ${ENV} 占位符）
│   └── Dockerfile
├── front/                   Vue 3 SPA
│   ├── src/
│   │   ├── components/      业务组件（K线、规则、自选……）
│   │   ├── pages/           路由页面（登录/首页）
│   │   ├── utils/api/       axios 封装
│   │   ├── router/
│   │   ├── App.vue
│   │   └── main.js
│   ├── vite.config.js
│   ├── package.json
│   ├── Dockerfile
│   └── nginx.conf
├── scripts/                 Python 采集/计算/调度
│   ├── config.ini           DB URL（占位）
│   ├── create_table.sql     全量 DDL（含 users、money_flow、target_trend_stock……）
│   ├── refresh_mv.sql       stock_history_mv 物化视图
│   ├── get_basic_info.py    股票基础信息
│   ├── get_stock_daily.py   日线
│   ├── get_money_flow_v2.py 资金流榜单
│   ├── get_financial_info.py 财报
│   ├── compute_indicators.py 技术指标（MA/MACD/KDJ/RSI/BOLL）
│   ├── refresh_mv.py        物化视图创建/刷新
│   ├── timer.py             调度器（16:00 起）
│   ├── pyproject.toml
│   └── requirements.txt
├── docs/                    库表设计文档
├── cache/                   离线缓存 CSV
├── docker-compose.yml
├── .env.example
└── README.md
```

## 接口列表（节选）

| Method | Path | 说明 | 鉴权 |
|---|---|---|---|
| POST | /api/v1/user/register | 注册 | 公开 |
| POST | /api/v1/user/login    | 登录 → 返 token | 公开 |
| GET  | /api/v1/user/favorites       | 自选股 | JWT |
| POST | /api/v1/user/favorites       | 添加自选 | JWT |
| DELETE | /api/v1/user/favorites/symbol/:symbol | 按股票代码取消自选 | JWT |
| POST | /api/v1/user/rules          | 新增选股规则 | JWT |
| GET  | /api/v1/user/rules          | 列出规则 | JWT |
| PUT  | /api/v1/user/rules/:id      | 修改 | JWT |
| DELETE | /api/v1/user/rules/:id    | 删除 | JWT |
| POST | /api/v1/user/rules/preview  | 预览规则（不入库） | JWT |
| POST | /api/v1/user/rules/:id/run  | 执行规则 → 写入 target_trend_stock | JWT |
| GET  | /api/v1/stocks/list         | 股票列表（分页） | 公开 |
| GET  | /api/v1/stocks/search?q=    | 模糊搜索 | 公开 |
| GET  | /api/v1/stocks/hot          | 热门（涨幅≥5%） | 公开 |
| GET  | /api/v1/stocks/history?symbol=&days= | 日线+指标+资金流 | 公开 |
| GET  | /api/v1/target-stocks?rule_name= | 候选股 | 公开 |

完整 OpenAPI 见 `http://localhost:3003/swagger/index.html`

## 选股规则表达式 (JSONB) 语法

```jsonc
{
  // 简单比较
  "change_percent": {"gt": 5, "lt": 9.8},    // > 5 AND < 9.8
  "turnover_rate":  {"gte": 3},             // >= 3
  "current_price":  {"between": [5, 50]},   // BETWEEN 5 AND 50

  // 集合
  "industry": {"in": ["银行", "证券"]},     // IN (...)
  "market":   "创业板",                       // =
  "symbol_prefix": "300",                    // LIKE '300%'

  // 统计（由 CTE 窗口函数实时算）
  "consecutive_up_days":        {"gte": 3},    // 连续 N 天上涨
  "consecutive_inflow_days":    {"gte": 3},    // 连续 N 天主力净流入
  "consecutive_volume_amplify_days": {"gte": 3}, // 连续 N 天放量
  "volume_amplify_days":        {"gte": 3, "min_ratio": 1.5} // 放量倍数
}
```

操作符：`gt` / `gte` / `lt` / `lte` / `eq` / `between` / `in`

## 路线图

- [x] 全量 DDL（10 张表 + 物化视图）
- [x] Go HTTP API + Swagger
- [x] HMAC 自实现 JWT
- [x] 选股规则执行（JSONB → CTE → target_trend_stock）
- [x] 技术指标自动计算
- [x] 物化视图自动刷新
- [x] Docker 一键启动
- [x] 前后端全打通（K 线 + 自选 + 规则 + 候选）
- [ ] 单元测试
- [ ] 实时推送（WebSocket）
- [ ] 回测
