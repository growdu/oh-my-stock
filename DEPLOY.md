# 部署与运行（systemd + nginx + backend 二进制）

## 一句话总结

- **前端**：nginx 直接 serve `front/dist`（热更时跑 `yarn dev` 用 :5713）
- **后端**：`backend/bin/oh-my-stock`，跑 systemd transient 或 `bash run-with-db.sh` 起在 :3003
- **DB**：本机 PostgreSQL `oh_my_stock` 库（启动前先跑 `scripts/create_table.sql`）
- **数据**：`systemd` 的两个 `.timer` 在 Mon-Fri 17:00 / 18:00 自动刷

> docker-compose 路径已废弃（`pgdata/` 是直接 host 上的目录，不再用容器）。

## 访问入口

| URL | 内容 | 备注 |
|---|---|---|
| `http://localhost/` | **生产入口**（nginx → front/dist + 3003） | 80 是 default_server |
| `http://localhost:18080/` | 同上的旧端口（兼容保留） | 用来跑前端的脚本还在用 |
| `http://localhost:5713/` | vite dev（热更） | `cd front && yarn dev` |
| `http://localhost:3003/` | backend 直连（无 nginx，绕过 SPA 兼容） | 开发 API 调试 |
| `http://localhost:8080/` | 旧项目 oh-my-cook | 已被挪到 8080 让出 80 |

> 你看到「打开全是报错 / 看不到数据」，90% 是把 `localhost:80` 误认为 oh-my-stock。
> 现在 80 已经让给 oh-my-stock（`sites-enabled/oh-my-stock.conf` 是 80 default_server），
> `oh-my-cook` 被搬到 8080 顶上。

## 启动顺序

```bash
# 1) PostgreSQL (systemd, host 上跑)
sudo systemctl start postgresql
# pgdata 在仓库根目录 pgdata/，端口 5432
PGPASSWORD='please_change_me_in_prod' psql -h 127.0.0.1 -U postgres -d oh_my_stock -c "\dt"

# 2) 后端 (./run-with-db.sh 把 DB env 写死，不需要 source .env)
cd /home/ubuntu/oh-my-stock/backend
bash run-with-db.sh    # 监听 :3003，写 logs/server.out

# 或用 systemd-run（不依赖终端，关 SSH 也不死）
systemd-run --user --unit=oh-my-stock-backend-dev \
  --working-directory=/home/ubuntu/oh-my-stock/backend \
  --setenv=DB_HOST=127.0.0.1 DB_PORT=5432 DB_USER=postgres \
  --setenv=DB_PASS='please_change_me_in_prod' DB_NAME=oh_my_stock \
  --setenv=JWT_SECRET=oh-my-stock-dev-secret \
  --setenv=SERVER_HOST=0.0.0.0 SERVER_PORT=3003 \
  /home/ubuntu/oh-my-stock/backend/bin/oh-my-stock

# 3) 前端 (两种，挑一种)
# (a) 生产 build（nginx 已经把 80/18080 指到 front/dist）
cd /home/ubuntu/oh-my-stock/front
node node_modules/vite/bin/vite.js build           # → dist/

# (b) 开发热更
node node_modules/vite/bin/vite.js                 # → 5713
```

## systemd 定时任务

```bash
bash deploy/systemd/install.sh      # 一键装 / 启用

sudo systemctl list-timers oh-my-stock-*           # 看下一次触发
sudo systemctl start oh-my-stock-daily-refresh.service   # 手动跑一次
sudo journalctl -u oh-my-stock-daily-refresh.service -e  # 看运行时日志
tail -f logs/daily_refresh_*.log                         # stdout
```

- **`oh-my-stock-daily-refresh.timer`** —— Mon-Fri 17:00，跑 `daily_refresh.sh`：基础信息 → K 线 → 资金流 → 财报 → 指标 → MV
- **`oh-my-stock-mv-fallback.timer`** —— Mon-Fri 18:00，单刷一次 `refresh_mv.py`，兜底

> A 股 15:00 收盘，17:00 是数据源（Sina / Eastmoney）当日 K 线 / 资金流最终化入库的稳定窗口；
> 18:00 再刷一次 MV 是为了 17:00 哪一步异常时前端不至于看到陈旧数据。

## 修改 / 重新构建

```bash
# 改 Go 代码
cd /home/ubuntu/oh-my-stock/backend
go build -o bin/oh-my-stock ./
# 重启
systemctl restart user/oh-my-stock-backend-dev       # 如果之前用 systemd-run

# 改前端
cd /home/ubuntu/oh-my-stock/front
node node_modules/vite/bin/vite.js build             # 重 build dist/
# reload nginx 让 80 拿到新 dist（或本身就直读文件，html 强制刷新就行）
```

## nginx 配置位置

`/etc/nginx/sites-enabled/oh-my-stock.conf` 和 `/etc/nginx/sites-enabled/default`：

- `oh-my-stock.conf` —— 本项目，**`:80 default_server`** + `:18080 default_server`，root 指向 `front/dist`，`/api/` reverse-proxy 到 `127.0.0.1:3003`
- `default` —— `oh-my-cook`，被挪到 `:8080 + server_name oh-my-cook.local`，让出 80

```bash
sudo nginx -t                 # 改完先测语法
sudo nginx -s reload          # 热重载
```

## 常见问题

**`relation "target_trend_stock" does not exist`**  
DB 没建表就启了 backend。`scripts/create_table.sql` 走一遍（DDL 全是 `IF NOT EXISTS`，可重复跑）。

**`bind: address already in use`**  
八成是上一次的 backend 没清干净。`pgrep -fa oh-my-stock`，`pkill` 后再起。

**前端的「网络错误」或 axios 超时**  
1. 先看 backend：`curl http://localhost:3003/healthz` 应回 `{"ok":true}`
2. nginx reload 是否成功：`sudo nginx -t && sudo nginx -s reload`
3. 前端的 baseURL 是 `/api/v1`，由 vite proxy / nginx 反代到 3003；如改 backend 端口，同时改 `front/vite.config.js` 里的 `VITE_API_BACKEND` 和 `sites-enabled/oh-my-stock.conf` 的 `proxy_pass`

**管理员忘记密码**  
ADMIN_USER/ADMIN_PASS 在 `.env`，启动时 backend 会自动 upsert；改完重启 backend 即可生效。
