"""
fetch_money_flow.py (v2)
========================
多端点 + 磁盘缓存 + 静默降级 的资金流 fetcher。

数据源（按顺序探测，单只失败自动级联）：
  1. Eastmoney push2his (主力接口，5 日窗口)
  2. Eastmoney push2 (备用主域)
  3. akshare stock_individual_fund_flow (内部走 datacenter-web.eastmoney.com)

容错策略：
  - 网络探测 → 全失败则直接退出 0 (不阻塞后续步骤)
  - 单只失败 → 跳过，继续下一只
  - 写库失败 → 报错但不影响其它记录
  - 增量刷新：DB 已有 (symbol, trade_date, time_span) 不重写 (用 UPSERT)
"""
import os, sys, time, json, logging, hashlib, configparser
from datetime import datetime, timedelta
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path

import requests, pandas as pd
from sqlalchemy import create_engine, text
from sqlalchemy.dialects.postgresql import insert

# ---------- DB ----------
cfg = configparser.ConfigParser()
cfg.read("config.ini", encoding="utf-8")
DB = cfg.get("database", "url")

CACHE_DIR = Path("logs/cache/money_flow")
CACHE_DIR.mkdir(parents=True, exist_ok=True)


def _cache_key(symbol: str, source: str) -> Path:
    h = hashlib.md5(f"{symbol}|{source}".encode()).hexdigest()
    return CACHE_DIR / f"{symbol}_{source}_{h}.json"


def _cache_get(key: Path, max_age_days: int = 2):
    if not key.exists():
        return None
    age = time.time() - key.stat().st_mtime
    if age > max_age_days * 86400:
        return None
    try:
        return json.loads(key.read_text(encoding="utf-8"))
    except Exception:
        return None


def _cache_put(key: Path, data):
    try:
        key.write_text(json.dumps(data, ensure_ascii=False, default=str),
                       encoding="utf-8")
    except Exception:
        pass


# ---------- HTTP ----------
def _new_session() -> requests.Session:
    s = requests.Session()
    s.headers["User-Agent"] = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) " \
                              "AppleWebKit/537.36 (KHTML, like Gecko) " \
                              "Chrome/120.0.0.0 Safari/537.36"
    s.headers["Referer"] = "https://quote.eastmoney.com/"
    s.trust_env = False
    return s


S_EM = _new_session()
S_EM_ALT = _new_session()
S_EM_ALT.headers["Referer"] = "https://data.eastmoney.com/"

URL_EM_DAYK = "https://push2his.eastmoney.com/api/qt/stock/fflow/daykline/get"
URL_EM_ALT  = "https://push2.eastmoney.com/api/qt/stock/fflow/daykline/get"


def _secid(sym: str) -> str:
    if sym.startswith(("60","68","90","11","13","5")):
        return f"1.{sym}"
    return f"0.{sym}"


def _try_one_url(url: str, secid: str, lmt: int, sess: requests.Session) -> list:
    for attempt in range(3):
        try:
            r = sess.get(url, params={
                "fields1": "f1,f2,f3,f4",
                "fields2": "f51,f52,f53,f54,f55,f56,f57,f58",
                "klt": 101, "lmt": lmt,
                "secid": secid,
            }, timeout=8)
            if r.status_code != 200:
                time.sleep(0.3 * (attempt + 1))
                continue
            try:
                d = r.json()
            except Exception:
                time.sleep(0.5)
                continue
            kls = (d.get("data") or {}).get("klines") or []
            rows = []
            for line in kls:
                p = line.split(",")
                if len(p) < 8:
                    continue
                rows.append({
                    "symbol": secid.split(".",1)[1],
                    "trade_date": datetime.strptime(p[0], "%Y-%m-%d").date(),
                    "main_net":   float(p[1]),
                    "retail_net": float(p[2]),
                    "mid_net":    float(p[3]),
                    "big_net":    float(p[4]),
                    "super_net":  float(p[5]),
                    "main_pct":   float(p[6]),
                    "turnover":   float(p[7]),
                })
            return rows
        except (requests.ConnectionError, requests.Timeout) as e:
            logging.debug("URL %s attempt %d: %s", url, attempt, e)
            time.sleep(0.5 * (2 ** attempt))
    return []


def fetch_em(sym: str, lmt: int = 5) -> list:
    secid = _secid(sym)
    rows = _try_one_url(URL_EM_DAYK, secid, lmt, S_EM)
    if rows:
        return rows
    return _try_one_url(URL_EM_ALT, secid, lmt, S_EM_ALT)


def fetch_ak(sym: str, lmt: int = 5) -> list:
    """兜底走 akshare（datacenter-web.eastmoney.com）"""
    try:
        import akshare as ak
    except ImportError:
        return []
    try:
        market = "sh" if sym.startswith(("60","68","90","11","13","5")) else "sz"
        df = ak.stock_individual_fund_flow(stock=sym, market=market)
        if df is None or df.empty:
            return []
        df = df.head(lmt).copy()
        rename = {
            "日期": "trade_date", "主力净流入-净额": "main_net",
            "主力净流入-净占比": "main_pct",
            "大单净流入-净额": "big_net",
            "特大单净流入-净额": "super_net",
            "中单净流入-净额": "mid_net",
            "小单净流入-净额": "retail_net",
        }
        # only keep columns that exist
        cols_in = {k: v for k, v in rename.items() if k in df.columns}
        df = df[list(cols_in.keys())].rename(columns=cols_in)
        df["symbol"] = sym
        df["turnover"] = 0.0
        if "trade_date" in df.columns:
            df["trade_date"] = pd.to_datetime(df["trade_date"]).dt.date
        # re-order to consistent keys
        for k in ["main_net","retail_net","mid_net","big_net","super_net","main_pct","turnover"]:
            if k not in df.columns: df[k] = 0.0
        return df.to_dict("records")
    except Exception as e:
        logging.debug("ak fail %s: %s", sym, e)
        return []


SOURCES = [
    ("eastmoney", fetch_em),
    ("akshare",   fetch_ak),
]


# ---------- 单只级联 ----------
def fetch_one(sym: str, lmt: int = 5, use_cache: bool = True) -> tuple:
    """返回 (source_name, rows)"""
    # 命中 cache 跳过
    if use_cache:
        cache_key = _cache_key(sym, "latest")
        cached = _cache_get(cache_key, max_age_days=2)
        if cached and cached.get("rows"):
            return cached.get("source", "cache"), cached["rows"]

    last_err = ""
    for src_name, fn in SOURCES:
        try:
            rows = fn(sym, lmt)
            if rows:
                _cache_put(_cache_key(sym, "latest"),
                           {"rows": rows, "source": src_name,
                            "ts": time.strftime("%Y-%m-%d %H:%M:%S")})
                return src_name, rows
        except Exception as e:
            last_err = str(e)[:100]
    if last_err:
        logging.debug("all sources failed for %s: %s", sym, last_err)
    return None, []


# ---------- 网络探测 ----------
def probe_sources(retries: int = 3, retry_wait: float = 5.0) -> list:
    """网络探测；任一源稳定 reachable 即返回。
    整体重试 retries 次（默认 3 次，每次间隔 retry_wait 秒），
    应对 eastmoney 偶发的 'Remote end closed connection' 瞬断。
    """
    sym = "600000"
    for attempt in range(1, retries + 1):
        reachable = []
        for src_name, fn in SOURCES:
            try:
                rows = fn(sym, 5)
                if rows:
                    logging.info("✓ %s reachable (%d rows)", src_name, len(rows))
                    reachable.append(src_name)
                else:
                    logging.warning("✗ %s replied empty", src_name)
            except Exception as e:
                logging.warning("✗ %s unreachable: %s", src_name, e)
        if reachable:
            if attempt > 1:
                logging.info("network recovered after %d attempt(s)", attempt)
            return reachable
        if attempt < retries:
            logging.info("probe failed (attempt %d/%d), retry in %.1fs ...",
                         attempt, retries, retry_wait)
            time.sleep(retry_wait)
    return []


# ---------- DB upsert ----------
def _flush(eng, rows):
    if not rows:
        return
    # time_span=0 表示日级
    placeholders = ",".join([f"'{r['trade_date']}'" for r in rows])
    with eng.begin() as conn:
        try:
            tdf = pd.read_sql(
                text(f"SELECT symbol, trade_date, turnover FROM stock_daily_data "
                     f"WHERE trade_date IN ({placeholders})"), eng)
            tmap = {(r.symbol, r.trade_date): float(r.turnover or 0)
                    for r in tdf.itertuples()}
        except Exception as e:
            logging.warning("读 stock_daily_data 失败: %s", e)
            tmap = {}
        params = [{
            "sym":   r["symbol"],
            "net":   r["main_net"],
            "to":    r["turnover"],
            "to_amt": tmap.get((r["symbol"], r["trade_date"]), 0),
            "dt":    r["trade_date"],
            # inflow / outflow = 大单 + 超大单 / 中单 + 小单 (eastmoney / akshare 都按此拆分)
            "in_amt": (r.get("big_net") or 0) + (r.get("super_net") or 0),
            "out_amt": (r.get("mid_net") or 0) + (r.get("retail_net") or 0),
        } for r in rows]
        # 分批
        BATCH = 500
        for i in range(0, len(params), BATCH):
            sql = text("""
                INSERT INTO stock_money_flow_all
                    (time_span, symbol, net_amount, turnover_rate, turnover,
                     inflow_amount, outflow_amount, trade_date)
                VALUES (0, :sym, :net, :to, :to_amt, :dt, :in_amt, :out_amt)
                ON CONFLICT (symbol, trade_date, time_span) DO UPDATE SET
                    net_amount     = EXCLUDED.net_amount,
                    turnover_rate  = EXCLUDED.turnover_rate,
                    turnover       = EXCLUDED.turnover,
                    inflow_amount  = EXCLUDED.inflow_amount,
                    outflow_amount = EXCLUDED.outflow_amount
            """)
            conn.execute(sql, params[i:i+BATCH])


def main():
    logging.basicConfig(level=logging.INFO,
                        format="%(asctime)s [%(levelname)s] %(message)s",
                        datefmt="%H:%M:%S")

    # 1. 网络探测
    reachable = probe_sources()
    if not reachable:
        logging.warning("⚠️  全部资金流接口在本机不可达，整步跳过（不影响 K 线 / 指标 / MV）")
        logging.warning("    缓存目录: %s", CACHE_DIR)
        sys.exit(0)
    logging.info("可用数据源: %s", reachable)

    # 2. 拉所有股票
    eng = create_engine(DB)
    syms = pd.read_sql(text("SELECT symbol FROM stock_basic_info"), eng)["symbol"].tolist()
    logging.info("[start] %d stocks, %d days each", len(syms), 5)

    src_stats = {s: 0 for s, _ in SOURCES}
    fail = 0
    rows_all = []
    with ThreadPoolExecutor(max_workers=4) as exe:
        futs = {exe.submit(fetch_one, s, 5): s for s in syms}
        for i, fut in enumerate(as_completed(futs), 1):
            src, rs = fut.result()
            if not rs:
                fail += 1
            else:
                src_stats[src] = src_stats.get(src, 0) + 1
                rows_all.extend(rs)
            if i % 500 == 0:
                logging.info("[%d/%d] fail=%d rows=%d src=%s",
                             i, len(syms), fail, len(rows_all), src_stats)
    if not rows_all:
        logging.warning("⚠️ 本次未拉到任何资金流（缓存里可能有历史数据可用）")
        sys.exit(0)

    logging.info("[flush] %d rows", len(rows_all))
    _flush(eng, rows_all)
    logging.info("[done] src=%s fail=%d", src_stats, fail)


if __name__ == "__main__":
    main()
