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

## 一次性初始化（按顺序执行）

```bash
python get_basic_info.py     # 拉全部 A 股基础信息 → stock_basic_info
python get_stock_daily.py    # 拉最新 3 个交易日日线 → stock_daily_data
python get_money_flow_v2.py  # 拉资金流榜单 → stock_money_flow_all
python get_financial_info.py # 拉财报 → stock_financial_data
python compute_indicators.py # 计算 MA/MACD/KDJ/RSI/BOLL → stock_indicators
python refresh_mv.py         # 创建/刷新 stock_history_mv 物化视图
```

## 定时任务

```bash
python timer.py    # 每天 16:00 起调度全部脚本
IMMEDIATE_RUN=1 python timer.py  # 调试时：先立即跑一次再调度
```

调度时刻：

| 时刻 | 脚本 |
|---|---|
| 16:00 | get_stock_daily |
| 16:05 | get_money_flow_v2 |
| 16:30 | get_financial_info |
| 17:00 | compute_indicators |
| 17:05 | refresh_mv |

## 物化视图

`refresh_mv.py` 调用 `refresh_mv.sql`：

- 首次：CREATE MATERIALIZED VIEW stock_history_mv + 唯一索引 + REFRESH MATERIALIZED VIEW
- 之后：REFRESH MATERIALIZED VIEW CONCURRENTLY stock_history_mv（唯一索引是前提）

## 注意事项

- AKShare 数据源不稳定，脚本均带 try/except + 时间戳日志
- 日志写到 `../logs/` 下，命名 `<时间戳>_<脚本名>.log`
- `compute_indicators.py` 对 < 60 条日线的股票会跳过
