"""
fetch_daily.py (v2)
===================
多源 + 指数退避 的 A 股 K 线 fetcher。

数据源（自动按顺序探测）：
  1. Tencent web.ifzq.gtimg.cn  —— 主，HTTP/2，前复权日 K
  2. Eastmoney push2his.eastmoney.com  —— 备，全历史后复权日 K
  3. Sina money.finance.sina.com.cn  —— 最后兜底（对部分 IP 被墙）

每只股票：对每个数据源最多 3 次重试，源之间自动级联。
启动时做一次网络探测，命中差源立即跳过下游，避免拖死整条管线。
"""
import os, sys, time, logging, configparser
from concurrent.futures import ThreadPoolExecutor, as_completed

import requests
import pandas as pd
from sqlalchemy import create_engine
from sqlalchemy.dialects.postgresql import insert

# ---------- DB ----------
cfg = configparser.ConfigParser()
cfg.read("config.ini", encoding="utf-8")
DB = cfg.get("database", "url")

# ---------- 请求会话 ----------
def _new_session(ua: str = "Mozilla/5.0", referer: str = "") -> requests.Session:
    s = requests.Session()
    s.headers["User-Agent"] = ua
    if referer:
        s.headers["Referer"] = referer
    s.trust_env = False
    return s

S_TX = _new_session(
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "
    "(KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
    "https://gu.qq.com/",
)
S_EM = _new_session(
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "
    "(KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
    "https://quote.eastmoney.com/",
)
S_SN = _new_session(
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "
    "(KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
    "https://finance.sina.com.cn/",
)

URL_TX = "https://web.ifzq.gtimg.cn/appstock/app/fqkline/get"
URL_EM = "https://push2his.eastmoney.com/api/qt/stock/kline/get"
URL_SN = "https://money.finance.sina.com.cn/quotes_service/api/json_v2.php/CN_MarketData.getKLineData"

# ---------- 工具 ----------
def prefix_of(symbol: str) -> str:
    if symbol.startswith(("60", "68", "90", "11", "13", "5")):
        return "sh"
    if symbol.startswith(("43", "83", "87", "88")):
        return "bj"
    return "sz"

def em_market(symbol: str) -> int:
    p = symbol[:2]
    if p in ("60", "68", "90", "11", "13"):  # 上交所
        return 1
    if p in ("43", "83", "87", "88"):        # 北交所
        return 0
    return 0                                # 深交所

def backoff_sleep(attempt: int):
    time.sleep(min(0.3 * (2 ** attempt), 6.0))


# ---------- 各数据源 fetcher ----------
def fetch_tencent(symbol: str, datalen: int = 300) -> list:
    p = prefix_of(symbol)
    url = URL_TX
    params = {"param": f"{p}{symbol},day,,,{datalen},qfq"}
    r = S_TX.get(url, params=params, timeout=8)
    if r.status_code != 200:
        return []
    try:
        data = r.json().get("data", {}).get(f"{p}{symbol}", {})
    except Exception:
        return []
    qfqday = data.get("qfqday") or []
    if not qfqday:
        return []
    rows = []
    prev_close = None
    from datetime import datetime
    # Tencent qfqday: [date, open, close, high, low, volume]
    for k in qfqday:
        try:
            d, o, c, h, l, vol = (k[0], k[1], k[2], k[3], k[4], k[5])
            op, cl = float(o), float(c)
            hi, lo = float(h), float(l)
            v = int(float(vol))
            change_amount = cl - prev_close if prev_close is not None else 0.0
            change_pct = (cl - prev_close) / prev_close * 100 if prev_close else 0.0
            amplitude = (hi - lo) / prev_close * 100 if prev_close else 0.0
            rows.append({
                "symbol": symbol,
                "trade_date": datetime.strptime(d, "%Y-%m-%d").date(),
                "open": op, "high": hi, "low": lo, "close": cl,
                "volume": v,
                "turnover": cl * v,
                "change_percent": change_pct,
                "change_amount": change_amount,
                "amplitude": amplitude,
                "turnover_rate": None,
            })
            prev_close = cl
        except (ValueError, IndexError, TypeError):
            continue
    return rows


def fetch_eastmoney(symbol: str, datalen: int = 300) -> list:
    secid = f"{em_market(symbol)}.{symbol}"
    url = URL_EM
    params = {
        "secid": secid,
        "klt": 101,        # 日 K
        "fqt": 1,          # 前复权
        "beg": 0,
        "end": 20500000,
        "fields1": "f1,f2,f3,f4",
        "fields2": "f51,f52,f53,f54,f55,f56,f57,f58,f59,f60,f61",
    }
    r = S_EM.get(url, params=params, timeout=8)
    if r.status_code != 200:
        return []
    try:
        kls = (r.json().get("data") or {}).get("klines") or []
    except Exception:
        return []
    if not kls:
        return []
    rows = []
    prev_close = None
    from datetime import datetime
    # Eastmoney klines CSV: date,open,close,high,low,volume,amount,振幅,涨跌幅,涨跌额,换手率
    for line in kls:
        p = line.split(",")
        if len(p) < 11:
            continue
        try:
            cl = float(p[2])
            op, hi, lo = float(p[1]), float(p[3]), float(p[4])
            v = int(float(p[5]))
            rows.append({
                "symbol": symbol,
                "trade_date": datetime.strptime(p[0], "%Y-%m-%d").date(),
                "open": op, "high": hi, "low": lo, "close": cl,
                "volume": v,
                "turnover": float(p[6]),
                "change_percent": float(p[8]),
                "change_amount": float(p[9]),
                "amplitude": float(p[7]),
                "turnover_rate": float(p[10]),
            })
            prev_close = cl
        except (ValueError, IndexError):
            continue
    return rows


def fetch_sina(symbol: str, datalen: int = 300) -> list:
    s = f"{prefix_of(symbol)}{symbol}"
    params = {"symbol": s, "scale": 240, "datalen": datalen}
    r = S_SN.get(URL_SN, params=params, timeout=8)
    txt = r.text.strip()
    if r.status_code != 200 or not txt.startswith("["):
        return []
    try:
        import json
        data = json.loads(txt)
    except Exception:
        return []
    if not data:
        return []
    rows = []
    prev_close = None
    from datetime import datetime
    for k in data:
        try:
            close = float(k["close"])
            rows.append({
                "symbol": symbol,
                "trade_date": datetime.strptime(k["day"], "%Y-%m-%d").date(),
                "open": float(k["open"]),
                "high": float(k["high"]),
                "low":  float(k["low"]),
                "close": close,
                "volume": int(k["volume"]),
                "turnover": float(k["close"]) * int(k["volume"]),
                "change_percent": (close - prev_close) / prev_close * 100 if prev_close else 0,
                "change_amount":  close - prev_close if prev_close else 0,
                "amplitude": (float(k["high"]) - float(k["low"])) / prev_close * 100 if prev_close else 0,
                "turnover_rate": None,
            })
            prev_close = close
        except (KeyError, ValueError):
            continue
    return rows


# 数据源优先级
SOURCES = [
    ("tencent",   fetch_tencent),
    ("eastmoney", fetch_eastmoney),
    ("sina",      fetch_sina),
]


# ---------- 单只：级联 + 指数退避 ----------
def fetch_one(symbol: str, datalen: int = 300) -> tuple:
    """返回 (source_name, rows). 没取到数据 → (None, [])"""
    for src_name, fn in SOURCES:
        for attempt in range(3):
            try:
                rows = fn(symbol, datalen)
                if rows:
                    return src_name, rows
            except requests.RequestException as e:
                logging.debug("%s/%s attempt %d: %s", src_name, symbol, attempt, e)
            backoff_sleep(attempt)
    return None, []


# ---------- 网络探测 ----------
def probe_sources() -> list:
    """按优先级对每源发一次请求，返回可达的源列表"""
    sym = "600000"
    reachable = []
    for src_name, fn in SOURCES:
        try:
            rows = fn(sym, 5)
            if rows:
                logging.info("✓ %s reachable (%d rows for %s)", src_name, len(rows), sym)
                reachable.append(src_name)
            else:
                logging.warning("✗ %s replied but empty", src_name)
        except Exception as e:
            logging.warning("✗ %s unreachable: %s", src_name, e)
    return reachable


# ---------- upsert ----------
def _upsert_method(table, conn, keys, data_iter):
    rows = list(data_iter)
    insert_stmt = insert(table.table).values(rows)
    update_cols = {c.name: insert_stmt.excluded[c.name]
                   for c in table.table.columns if c.name not in ("id",)}
    upsert = insert_stmt.on_conflict_do_update(
        index_elements=["symbol", "trade_date"],
        set_=update_cols)
    conn.execute(upsert)


def main():
    logging.basicConfig(level=logging.INFO,
                        format="%(asctime)s [%(levelname)s] %(message)s",
                        datefmt="%H:%M:%S")

    # 启动探测
    reachable = probe_sources()
    if not reachable:
        logging.error("❌ 全部数据源不可用，今天的 K 线任务退出（其它步骤不受影响）")
        sys.exit(0)
    logging.info("可用数据源: %s", reachable)

    eng = create_engine(DB, pool_size=10, max_overflow=20)
    syms = pd.read_sql("SELECT symbol FROM stock_basic_info ORDER BY symbol",
                       eng)["symbol"].tolist()
    logging.info("[start] %d stocks", len(syms))

    src_stats = {s: 0 for s, _ in SOURCES}
    fail = 0
    total = 0
    BATCH = 200
    batch = []

    def _flush():
        nonlocal total, batch
        if not batch:
            return
        df = pd.DataFrame(batch)
        df.to_sql("stock_daily_data", eng, if_exists="append",
                  index=False, method=_upsert_method)
        total += len(batch)
        batch = []

    with ThreadPoolExecutor(max_workers=12) as exe:
        futs = {exe.submit(fetch_one, s, 300): s for s in syms}
        for i, fut in enumerate(as_completed(futs), 1):
            sym = futs[fut]
            src, rows = fut.result()
            if not rows:
                fail += 1
            else:
                src_stats[src] = src_stats.get(src, 0) + 1
                batch.extend(rows)
                if len(batch) >= BATCH:
                    _flush()
            if i % 200 == 0 or i == len(syms):
                logging.info("[%d/%d] wrote=%d fail=%d src=%s",
                             i, len(syms), total, fail, src_stats)
    _flush()

    src_summary = " ".join(f"{k}={v}" for k, v in src_stats.items() if v)
    logging.info("[done] wrote=%d fail=%d | %s", total, fail, src_summary)
    if fail > len(syms) * 0.5:
        logging.warning("⚠️ 超过 50%% 股票拉取失败，请检查网络 / 数据源")


if __name__ == "__main__":
    main()
