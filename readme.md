# oh-my-stock

A 股实时行情 / 选股 / 通知的全栈示例：Go (Gin + GORM) + PostgreSQL + Vue 3 + Docker Compose。

## 架构

```
[ browser ] ──HTTP── [ nginx:8888 ] ──/api── [ backend:3003 (Go) ] ──SQL── [ postgres:16 ]
                                  └─SPA───────────────────────────
                          fetcher ──HTTP── 新浪行情中心 / 新浪日K / 东方财富 detail/估值表
```

- `backend/`  Go 1.23 + Gin + GORM，自带 jobs 定时增量抓数 + admin 维护端点。
- `front/`    Vue 3 + Element Plus + ECharts + Vite。
- `deploy/`   `docker-compose.yml` + Postgres 初始化 SQL + 运维脚本。

## 快速启动

```bash
cd deploy
cp .env.example .env
./start.sh          # 构建 + 启动，~ 1-2 分钟
# 前端 http://localhost:8888
# 后端 http://localhost:3003
# Swagger http://localhost:3003/swagger/index.html
./logs.sh           # 实时日志
```

默认管理员：`admin` / `admin123`（首次登录后请修改）。

## API 约定

所有响应包一层 axios 包装，所以前端读 `res.data.data` 才拿到真正的列表。

```jsonc
// 分页列表
{ "page": 1, "page_size": 20, "total": 1234, "data": [ ... ] }

// 错误
{ "error": "message" }
```

需要登录的接口要求 `Authorization: Bearer <token>`，从 `/api/v1/user/login` 获取。

## 主要端点

| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| POST | `/api/v1/user/register` | - | 注册 |
| POST | `/api/v1/user/login` | - | 登录，返回 `{user_id, token}` |
| GET  | `/api/v1/stocks/list` | - | 行情列表（来自 `stock_history_mv`） |
| GET  | `/api/v1/stocks/screen` | - | 多条件筛选（行业/价格/涨跌/PE/PB/资金） |
| GET  | `/api/v1/stocks/ranking?rank_by=&order=&limit=` | - | 排行榜（`rank_by`: change_percent/volume/turnover_rate/net_amount，`order`: desc/asc，默认 desc） |
| GET  | `/api/v1/stocks/industries` `/markets` | - | 筛选项 |
| GET  | `/api/v1/stocks/presets` | - | 列出内置选股预设 |
| GET  | `/api/v1/stocks/presets/:id/run?page=&page_size=` | - | 运行指定预设，返回命中股票列表 |
| POST | `/api/v1/stocks/presets/run-expression` | - | 直接传 JSON 表达式跑一次（用于自定义规则预览） |
| POST | `/api/v1/user/favorites` `DELETE /:symbol` | ✓ | 自选股 |
| GET  | `/api/v1/user/favorites?page=&page_size=` | ✓ | 自选股分页（symbol + created_at） |
| GET/POST/PUT/DELETE | `/api/v1/user/rules[/:id]` | ✓ | 选股规则 |
| GET  | `/api/v1/notifications?page=&page_size=` | ✓ | 当前用户通知分页 |
| GET  | `/api/v1/notifications/unread` | ✓ | 当前用户未读条数 |
| PUT  | `/api/v1/notifications/:id/read` `PUT /read-all` | ✓ | 标记已读 |
| POST | `/api/v1/notifications/check-rules` | ✓ | 立即跑当前用户规则并写通知 |
| POST | `/api/v1/admin/stock-basics/refetch-all` | X-Admin-Token | 触发全量补全 |
| POST | `/api/v1/admin/stock-basics/refetch-by-symbols` | X-Admin-Token | body `{symbols:[...]}` 同步补全指定股票 |
| POST | `/api/v1/admin/stock-daily/refetch-all` | X-Admin-Token | 触发全量日 K 抓取 |
| GET  | `/api/v1/admin/jobs/:id` | X-Admin-Token | 查询后台 job 状态 |
| POST | `/api/v1/admin/notifications/run-rules` | X-Admin-Token | 触发规则匹配写通知；body 可选 `{user_id}` 指定单个用户 |
| GET  | `/api/v1/admin/notifications/preview?user_id=X` | X-Admin-Token | 规则 dry-run，返回命中但不写通知 |
| GET  | `/api/v1/stocks/history?symbol=&days=` | - | 单股最近 N 日 K + 指标 + 资金流 |
| GET  | `/api/v1/stocks/search?q=` | - | 模糊搜索 |
| POST | `/api/v1/admin/stock-basics/refetch-all` | X-Admin-Token | 触发全量补全 |
| POST | `/api/v1/admin/stock-daily/refetch-all` | X-Admin-Token | 触发全量日 K 抓取 |
| GET  | `/api/v1/admin/jobs/:id` | X-Admin-Token | 查询后台 job 状态 |
| POST | `/api/v1/admin/notifications/run-rules` | X-Admin-Token | 触发规则匹配写通知；body 可选 `{user_id}` 指定单个用户 |
| GET  | `/api/v1/admin/notifications/preview?user_id=X` | X-Admin-Token | 规则 dry-run，返回命中但不写通知 |

## 开发

```bash
# 后端
cd backend
go build -mod=vendor ./...
go vet -mod=vendor ./...

# 前端
cd front
yarn install
yarn dev    # http://localhost:5713 ，代理 /api → 后端 3003
yarn build
```

环境变量（后端）：

| 变量 | 默认 | 说明 |
|------|------|------|
| `DB_HOST` `DB_PORT` `DB_USER` `DB_PASS` `DB_NAME` | postgres / 5432 / postgres / postgres / a_stock | Postgres 连接 |
| `FRONT_ORIGIN` | `*` | CORS 允许来源 |
| `ADMIN_TOKEN` | `oh-my-stock-admin` | `/admin/*` 路由所需 `X-Admin-Token` |
| `JWT_SECRET` | dev fallback | 登录 token 签名密钥，**生产必须改** |
