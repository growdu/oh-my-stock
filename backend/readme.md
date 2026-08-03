# 后端 (oh-my-stock API)

## 路由

参见 `/swagger/index.html`

## 本地运行

```bash
cp .env.example .env
# 改 DB_HOST / DB_PASS / JWT_SECRET ...
export $(grep -v '^#' .env | xargs)
go mod download
go run .
```

## 环境变量

| 变量 | 说明 |
|---|---|
| `DB_HOST` `DB_PORT` `DB_USER` `DB_PASS` `DB_NAME` | PostgreSQL 连接 |
| `JWT_SECRET`  | HMAC 签名密钥（必须 16 字节以上） |
| `JWT_TTL_HOURS` 默认 168 (7 天) |
| `SERVER_PORT` 默认 3003 |
| `FRONTEND_ORIGIN` 允许跨域的前端 origin（`http://localhost:5173`） |
| `CONFIG_PATH` 默认 `config.json` |

## 鉴权机制

- `/api/v1/user/login` → 返回 `token = base64url(payload).hex(hmac_sha256(payload))`
- payload = `{"uid": <uuid>, "iat": <unix_ts>, "exp": <unix_ts>}`
- 所有 `user/*` 路由要求 `Authorization: Bearer <token>`
- 失败统一 401 + 自动从 localStorage 清 token（前端已处理）

## 业务接口关键点

- `/stocks/list` / `/stocks/hot` / `/stocks/info` 都基于物化视图 `stock_history_mv`
- 物化视图由 `scripts/refresh_mv.py` 维护
- `/user/rules/:id/run` 把 `user_stock_rules.rule_expression` JSONB 翻译成 SQL（窗口函数 CTE），结果写入 `target_trend_stock`
