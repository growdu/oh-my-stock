"""
fetch_financial_lite.py — 拉年报/季报核心三指标 (净利润/营收/同比)。

接口（无需鉴权）：
  GET https://datacenter.eastmoney.com/securities/api/data/get
      ?type=RPT_F10_FINANCE_MAINFINADATA
      &filter=(SECUCODE="{code}.{market}")
      &ps=6&p=1&st=REPORT_DATE&sr=-1&source=HSF10&client=PC

顺序执行（约 6 分钟跑完全市场 5500 只），刷写一次性 commit。
每只股票最多保留最近 6 期（约 1.5 年）。
建议每季报披露期（4/8/10 月）跑一次。
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
        except Exception as e:
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

def main():
    eng = create_engine(DB)
    with eng.begin() as conn:
        syms = pd.read_sql("SELECT symbol FROM stock_basic_info", conn)["symbol"].tolist()
    print(f"[start] {len(syms)} stocks, 6 reports each (sequential)", flush=True)

    rows = []
    fail = 0
    t0 = time.time()
    for i, s in enumerate(syms, 1):
        r = fetch_one(s)
        if not r:
            fail += 1
        else:
            rows.extend(r)
        if i % 5 == 0 or i == len(syms):
            elapsed = time.time() - t0
            eta = elapsed / i * (len(syms) - i)
            print(f"  [{i}/{len(syms)}] rows={len(rows)} fail={fail} "
                  f"elapsed={elapsed:.0f}s eta={eta:.0f}s", flush=True)

    print(f"[flush] {len(rows)} rows from {len(syms)-fail} stocks ({fail} failed)", flush=True)
    with eng.begin() as conn:
        for i in range(0, len(rows), 1000):
            conn.execute(UPSERT_SQL, rows[i:i+1000])
    print(f"[done] upserted {len(rows)} rows in {time.time()-t0:.0f}s", flush=True)

if __name__ == "__main__":
    main()
