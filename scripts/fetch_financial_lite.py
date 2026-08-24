"""
fetch_financial_lite.py — 拉年报/季报核心三指标 (净利润/营收/同比)。

接口（无需鉴权）：
  GET https://datacenter.eastmoney.com/securities/api/data/get
      ?type=RPT_F10_FINANCE_MAINFINADATA
      &filter=(SECUCODE="{code}.{market}")
      &ps=6&p=1&st=REPORT_DATE&sr=-1&source=HSF10&client=PC

每只股票最多保留最近 6 期（约 1.5 年）。建议每季报披露期（4/8/10 月）跑一次。
每 200 只为一个 chunk，立即 UPSERT 入库，中途被杀也不会丢前面的数据。
"""
import configparser
import sys
import time
import requests
import pandas as pd
from datetime import datetime
from sqlalchemy import create_engine, text

cfg = configparser.ConfigParser()
cfg.read("config.ini", encoding="utf-8")
DB = cfg.get("database", "url")

URL = "https://datacenter.eastmoney.com/securities/api/data/get"
HEADERS = {
    "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "
                  "(KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
    "Referer": "https://emweb.securities.eastmoney.com/",
}

def to_num(v):
    if v is None or v == "" or v in ("-", "--"):
        return None
    try: return float(v)
    except: return None

def probe_endpoint() -> bool:
    """先拿一只测试股票试一次；不能则本机出口对该接口被掐，本步直接跳过。"""
    sym = "600000"
    for _ in range(2):
        try:
            r = requests.get(URL, params={
                "type": "RPT_F10_FINANCE_MAINFINADATA",
                "sty": "ALL",
                "filter": f"(SECUCODE=\"" + sym + ".SH\")",
                "p": "1", "ps": "6",
                "st": "REPORT_DATE", "sr": "-1",
                "source": "HSF10", "client": "PC",
            }, headers=HEADERS, timeout=8)
            d = r.json()
            if d.get("success") and (d.get("result", {}).get("data") or []):
                return True
        except Exception:
            pass
        time.sleep(0.5)
    return False

def secid_of(sym: str) -> tuple[str, str]:
    """返回 (code, market) → ('600519', 'SH')"""
    if sym.startswith(("60", "68", "90", "11", "13", "5")):
        return sym, "SH"
    if sym.startswith(("43", "83", "87", "88", "92")):
        return sym, "BJ"
    return sym, "SZ"

def fetch_one(sym: str) -> list:
    code, market = secid_of(sym)
    secucode = f"{code}.{market}"
    params = {
        "type": "RPT_F10_FINANCE_MAINFINADATA",
        "sty": "ALL",
        "filter": f"(SECUCODE=\"{secucode}\")",
        "p": "1",
        "ps": "6",
        "st": "REPORT_DATE",
        "sr": "-1",
        "source": "HSF10",
        "client": "PC",
    }
    for attempt in range(2):
        try:
            r = requests.get(URL, params=params, headers=HEADERS, timeout=10)
            d = r.json()
            if not d.get("success"):
                return []
            data = d.get("result", {}).get("data", []) or []
            rows = []
            for item in data:
                rdate = (item.get("REPORT_DATE", "") or "")[:10]
                if not rdate:
                    continue
                rows.append({
                    "symbol":         sym,
                    "report_date":    datetime.strptime(rdate, "%Y-%m-%d").date(),
                    "report_type":    item.get("REPORT_TYPE", ""),
                    "net_profit":     to_num(item.get("PARENTNETPROFIT")),
                    "total_revenue":  to_num(item.get("TOTALOPERATEREVE")),
                    "net_profit_yoy": to_num(item.get("PARENTNETPROFITTZ")),
                    "revenue_yoy":    to_num(item.get("TOTALOPERATEREVETZ")),
                })
            return rows
        except Exception:
            if attempt == 0:
                time.sleep(0.5)
                continue
            return []
    return []

UPSERT_SQL = text("""
    INSERT INTO stock_financial_data
        (symbol, report_date, report_type, net_profit, total_revenue,
         net_profit_yoy, revenue_yoy)
    VALUES
        (:symbol, :report_date, :report_type, :net_profit, :total_revenue,
         :net_profit_yoy, :revenue_yoy)
    ON CONFLICT (symbol, report_date, report_type) DO UPDATE SET
        net_profit     = EXCLUDED.net_profit,
        total_revenue  = EXCLUDED.total_revenue,
        net_profit_yoy = EXCLUDED.net_profit_yoy,
        revenue_yoy    = EXCLUDED.revenue_yoy
""")

CHUNK = 200  # 每 200 只一个 chunk，立即入库

def main():
    if not probe_endpoint():
        print("⚠️  财报接口 datacenter.eastmoney.com 在本机不可达，本步骤跳过。", flush=True)
        print("    K 线 / 指标 / MV 不受影响；如需恢复，排查出口 IP / 反代 / VPN。", flush=True)
        return
    eng = create_engine(DB)
    with eng.begin() as conn:
        syms = pd.read_sql("SELECT symbol FROM stock_basic_info", conn)["symbol"].tolist()
    print(f"[start] {len(syms)} stocks, chunk={CHUNK}", flush=True)

    t0 = time.time()
    rows_buf = []
    fail = 0
    done = 0
    for i, s in enumerate(syms, 1):
        r = fetch_one(s)
        if not r:
            fail += 1
        else:
            rows_buf.extend(r)

        # chunk 满了就入库
        if len(rows_buf) >= CHUNK * 6 or i % CHUNK == 0:
            with eng.begin() as conn:
                for k in range(0, len(rows_buf), 1000):
                    conn.execute(UPSERT_SQL, rows_buf[k:k+1000])
            done += CHUNK if i % CHUNK == 0 else 0
            elapsed = time.time() - t0
            eta = elapsed / i * (len(syms) - i)
            print(f"  [{i}/{len(syms)}] flushed={len(rows_buf)} fail={fail} "
                  f"elapsed={elapsed:.0f}s eta={eta:.0f}s", flush=True)
            rows_buf = []

    # flush 残余
    if rows_buf:
        with eng.begin() as conn:
            for k in range(0, len(rows_buf), 1000):
                conn.execute(UPSERT_SQL, rows_buf[k:k+1000])
    print(f"[done] {len(syms)-fail} ok, {fail} fail, total {time.time()-t0:.0f}s", flush=True)

if __name__ == "__main__":
    main()
