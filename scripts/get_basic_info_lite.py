"""
精简版 basic_info 拉取：只拉代码+名称，按代码前缀填 market。
industry 字段留空（16 个预设里没用 industry_in/not_in 条件）。
"""
import os, time, requests
import configparser
import pandas as pd
import akshare as ak
from sqlalchemy import create_engine, Column, Integer, String, Date, Boolean, DECIMAL, TIMESTAMP, text
from sqlalchemy.orm import sessionmaker, declarative_base
from datetime import datetime

cfg = configparser.ConfigParser()
cfg.read("config.ini", encoding="utf-8")
DB = cfg.get("database", "url")

# 给 akshare 加 UA
ak.requests = requests.Session()
ak.requests.headers.update({
    "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "
                  "(KHTML, like Gecko) Chrome/120.0 Safari/537.36"
})

Base = declarative_base()

class StockBasicInfo(Base):
    __tablename__ = "stock_basic_info"
    id = Column(Integer, primary_key=True, autoincrement=True)
    symbol = Column(String(10), unique=True, nullable=False)
    name = Column(String(50), nullable=False)
    full_name = Column(String(100))
    industry = Column(String(50))
    area = Column(String(50))
    market = Column(String(20))
    listing_date = Column(Date)
    outstanding_shares = Column(DECIMAL(20,4))
    total_shares = Column(DECIMAL(20,4))
    is_hs = Column(Boolean)
    status = Column(String(20))
    created_at = Column(TIMESTAMP, server_default=text('CURRENT_TIMESTAMP'))
    updated_at = Column(TIMESTAMP, server_default=text('CURRENT_TIMESTAMP'), onupdate=datetime.now)

def market_of(code: str) -> str:
    if code.startswith("68"): return "科创板"
    if code.startswith("60") or code.startswith("90"): return "主板-沪市"
    if code.startswith("00") or code.startswith("20"): return "主板-深市"
    if code.startswith("30"): return "创业板"
    if code.startswith("8") or code.startswith("92"): return "北交所"
    if code.startswith("43"): return "北交所"
    return "其他"

def main():
    eng = create_engine(DB)
    Sess = sessionmaker(bind=eng); s = Sess()

    print("[1/3] 拉取 A 股全量代码+名称 ...")
    df = ak.stock_info_a_code_name()
    df["code"] = df["code"].astype(str).str.zfill(6)
    print(f"      拉取到 {len(df)} 条")

    # 已存在的 symbol
    existing = set(pd.read_sql("SELECT symbol FROM stock_basic_info", eng)["symbol"].tolist())
    print(f"[2/3] 已存在 {len(existing)} 条")

    new, upd = 0, 0
    for _, row in df.iterrows():
        sym = row["code"]; name = row["name"]
        if not sym.isdigit() or len(sym) != 6:
            continue
        m = market_of(sym)
        obj = s.query(StockBasicInfo).filter_by(symbol=sym).first()
        if obj:
            obj.name = name
            obj.market = m
            obj.status = "上市"
            upd += 1
        else:
            s.add(StockBasicInfo(symbol=sym, name=name, market=m, status="上市"))
            new += 1
        if (new + upd) % 200 == 0:
            s.commit()
            print(f"      进度: +{new} 新增, ~{upd} 更新")

    s.commit()
    print(f"[3/3] 完成: 新增 {new}, 更新 {upd}")
    # 统计
    out = pd.read_sql("SELECT market, COUNT(*) c FROM stock_basic_info GROUP BY market ORDER BY c DESC", eng)
    print(out.to_string(index=False))
    s.close()

if __name__ == "__main__":
    main()
