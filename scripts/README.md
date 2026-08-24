# scripts · 数据采集/计算/调度

## 安装依赖

```bash
python -m venv .venv && source .venv/bin/activate  # Windows: .venv\Scripts\activate
pip install -r requirements.txt
```

## 配置

```bash
cp config.ini config.local.ini  # 修改 url
# 或者设置 DATABASE_URL 环境变量，部分脚本也识别
```

`config.ini` 示例：
```ini
[database]
url = postgresql://postgres:yourpass@localhost:5432/oh_my_stock
```

## 一键日刷管道（推荐入口）

```bash
./daily_refresh.sh               # 默认：拉 K 线 → 资金流 → 财报 → 算指标 → 刷 MV
./daily_refresh.sh --initial     # 首次部署：再加 basic_info 全量
./daily_refresh.sh --skip-fetch  # 只算指标 + 刷 MV (K 线已就位时)
./daily_refresh.sh --skip-compute
./daily_refresh.sh --skip-mv
```

## 定时调度

由 systemd .timer 接管（推荐，开机自启）：

```bash
bash deploy/systemd/install.sh   # 一键装上 two timers
sudo systemctl list-timers oh-my-stock-*
```

调度时刻 (Mon-Fri)：

| 时刻 | 服务 / 脚本 |
|---|---|
| 17:00 | `oh-my-stock-daily-refresh.service`  → `daily_refresh.sh` (拉+算+刷 MV 全套) |
| 18:00 | `oh-my-stock-mv-fallback.service`    → `refresh_mv.py` (兜底) |

`timer.py` 仅作为应急备选（`IMMEDIATE_RUN=1 python timer.py` 可立即跑一遍）。

## 物化视图

`refresh_mv.py` 调用 `refresh_mv.sql`：

- 首次：CREATE MATERIALIZED VIEW stock_history_mv + 唯一索引 + REFRESH MATERIALIZED VIEW
- 之后：REFRESH MATERIALIZED VIEW CONCURRENTLY stock_history_mv（唯一索引是前提）

## 注意事项

- AKShare 数据源不稳定，脚本均带 try/except + 时间戳日志
- 日志写到 `../logs/` 下，命名 `<时间戳>_<脚本名>.log`
- `compute_indicators.py` 对 < 60 条日线的股票会跳过
