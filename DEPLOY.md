# 部署与运行

## 一键脚本

```bash
./deploy.sh start     # 首次启动（自动生成 .env、构建镜像、初始化物化视图）
./deploy.sh stop      # 停止
./deploy.sh restart   # 重启
./deploy.sh status    # 健康检查
./deploy.sh logs      # 实时日志
./deploy.sh rebuild   # 强制重建镜像
```

## 首次部署步骤

```bash
# 1. 准备环境变量
cp .env.example .env
# 修改 DB_PASS / JWT_SECRET 等

# 2. 一键启动
./deploy.sh start
```

脚本会：
1. 自动生成 `.env`（若不存在）
2. `docker compose up -d --build` 启动 pg + backend + frontend
3. 等 `pg_isready` 通过
4. `docker compose up init-mv` 跑 `scripts/refresh_mv.sql`（创建 `stock_history_mv`）

## 端口

| 服务 | 端口 | 备注 |
|---|---|---|
| PostgreSQL | 5432 | 内网可用 `localhost:5432` |
| 后端 | 3003 | `/healthz`、`/api/v1/...` |
| 前端 | 5173 | nginx 反代 `/api/ → http://backend:3003/api/` |

## 目录持久化

- `pgdata`（docker volume）— PostgreSQL 数据
- 容器内 `/app/docs` — Swagger 静态资源
- `./scripts/create_table.sql` 在首次启动 PG 时自动跑（DDL 幂等）

## 数据采集（可选）

服务起来后，还需要数据：

```bash
cd scripts
python -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt

python get_basic_info.py     # 拉 A 股基础信息
python get_stock_daily.py    # 拉日线
python get_money_flow_v2.py  # 拉资金流
python compute_indicators.py # 计算指标
python refresh_mv.py         # 刷新物化视图

# —— 定时调度由 systemd 接管 ——
# (Mon-Fri 17:00 全量刷；18:00 单刷 MV 兜底)
bash deploy/systemd/install.sh
```

详见 [§ systemd 定时调度](#systemd-定时调度)。

## 验证

```bash
curl http://localhost:3003/healthz
# → {"ok":true}

# 注册 + 登录
curl -X POST http://localhost:3003/api/v1/user/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"demo","password":"demo1234"}'

curl -X POST http://localhost:3003/api/v1/user/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"demo","password":"demo1234"}'
# → 拿到 token，后续 user/* 接口加 Authorization: Bearer <token>
```


## systemd 定时调度

把 17:00 主刷 + 18:00 MV 兜底交给 systemd（开机自启，错过自动补）：

```bash
bash deploy/systemd/install.sh
```

实际做了什么：

1. `cp deploy/systemd/oh-my-stock-*.{service,timer} /etc/systemd/system/`
2. `systemctl daemon-reload`
3. `enable --now oh-my-stock-daily-refresh.timer` 和 `oh-my-stock-mv-fallback.timer`

日常使用：

```bash
sudo systemctl list-timers oh-my-stock-*            # 下次触发时刻
sudo systemctl start oh-my-stock-daily-refresh.service  # 立刻手动跑一次
sudo journalctl -u oh-my-stock-daily-refresh.service -e  # 看运行日志
tail -f logs/daily_refresh_systemd.log                   # 或 stdout 文件
```

替换以前的 `nohup python3 scripts/timer.py &`，后者在 shell 退出 / 机器重启后会丢，留下的定时进程会一直 crash-and-restart 不可靠。

时间设计：

- **17:00** 是 A 股 15:00 收盘后的「数据源稳定窗口」（Sina / Eastmoney 通常 17:00 前把当日 K 线 + 资金流最终化入库）
- **18:00** 是兜底再刷一次 MV，免得 17:00 整条管道哪一步异常时前端看到陈旧数据
- `Mon..Fri`——周末 / 节假日 A 股不开市，触发也只是空跑

## 容器自启策略

`docker-compose.yml` 已为 `postgres`、`backend`、`frontend` 设置 `restart: unless-stopped`。
- 机器重启 → 自动起
- 进程崩溃 → 自动重启
- 主动 `docker compose down` → 不会自动起

## 关闭/清理

```bash
./deploy.sh stop         # 停容器，保留数据卷
docker compose down -v   # 停容器并删除 pgdata
```
