import os
from datetime import datetime, timedelta
import pandas as pd
import akshare as ak
from decimal import Decimal
from sqlalchemy import create_engine, Column, String, Date, DECIMAL, BIGINT, Integer, UniqueConstraint, TIMESTAMP
from sqlalchemy.orm import sessionmaker, declarative_base
from sqlalchemy.exc import IntegrityError
import configparser
import requests
import time

# 给 requests 增加默认 headers
ak.session = requests.Session()
ak.session.headers.update({
    "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "
                  "AppleWebKit/537.36 (KHTML, like Gecko) "
                  "Chrome/120.0.0.0 Safari/537.36",
    "Accept-Language": "zh-CN,zh;q=0.9"
})

# ========== 配置区 ==========
# ========== 读取配置 ==========
config = configparser.ConfigParser()
config.read("config.ini", encoding="utf-8")
DATABASE_URL = config.get("database", "url")

# ========== ORM映射 ==========
Base = declarative_base()

class StockDailyData(Base):
    __tablename__ = "stock_daily_data"
    id = Column(Integer, primary_key=True)
    symbol = Column(String(10), nullable=False)
    trade_date = Column(Date, nullable=False)
    open = Column(DECIMAL(12,4))
    high = Column(DECIMAL(12,4))
    low = Column(DECIMAL(12,4))
    close = Column(DECIMAL(12,4))
    adj_close = Column(DECIMAL(12,4))
    volume = Column(BIGINT)
    turnover = Column(DECIMAL(20,4))
    change_percent = Column(DECIMAL(10,4))
    change_amount = Column(DECIMAL(10,4))
    turnover_rate = Column(DECIMAL(10,4))
    pe_ttm = Column(DECIMAL(10,4))
    pb = Column(DECIMAL(10,4))
    amplitude = Column(DECIMAL(10,4))
    created_at = Column(TIMESTAMP, default=datetime.now)

    __table_args__ = (
        UniqueConstraint('symbol', 'trade_date', name='uk_stock_daily'),
    )

# ========== 工具函数 ==========

def get_trade_dates():
    today_str = datetime.now().strftime("%Y-%m-%d")
    
    # 获取所有交易日
    df = ak.tool_trade_date_hist_sina()
    df["trade_date"] = df["trade_date"].astype(str)

    # 只保留今天及之前的交易日
    trade_dates = df[df["trade_date"] <= today_str]["trade_date"].tolist()

    # 取最近 30 个交易日
    last_15_days = trade_dates[-30:]
    
    return last_15_days

def is_trade_day(date: datetime, trade_dates: list) -> bool:
    return date.strftime("%Y%m%d") in trade_dates

def get_existing_trade_dates(session, symbol):
    rows = session.query(StockDailyData.trade_date).filter(StockDailyData.symbol == symbol).all()
    return set(r[0] for r in rows)

def fetch_and_store_stock_daily(session, symbol, start_date=None, end_date=None): 
    if len(symbol) < 6:
        print(f"unknown symbol:{symbol}")
        return
    existing_dates = get_existing_trade_dates(session, symbol)
    if start_date is None:
        start_date = datetime.now().strftime('%Y%m%d')
    else:
        start = datetime.strptime(start_date, "%Y-%m-%d").date()
        if start in existing_dates:
            print(symbol + "has imported")
            return
        start_date=pd.to_datetime(start_date).strftime('%Y%m%d')
    if end_date is None:
        end_date = datetime.now().strftime('%Y%m%d')
    else:
        end_date = pd.to_datetime(end_date).strftime('%Y%m%d')

    try:
        #df = ak.stock_zh_a_daily(symbol=symbol, start_date=start_date, end_date=end_date, adjust="qfq")
        time.sleep(1)
        df = ak.stock_zh_a_hist(symbol=symbol, period="daily", adjust="qfq", start_date=start_date,timeout=10)
    except Exception as e:
        print(f"获取{symbol}日线数据失败: {e}")
        return

    if df.empty:
        print(f"{symbol}无日线数据")
        return

    for _, row in df.iterrows():
        trade_date = row['日期']
        if trade_date in existing_dates:
            continue

        record = StockDailyData(
            symbol=symbol,
            trade_date=trade_date,
            open=row.get('开盘'),
            high=row.get('最高'),
            low=row.get('最低'),
            close=row.get('收盘'),
            adj_close=row.get('后复权收盘'),
            volume=row.get('成交量'),
            turnover=row.get('成交额'),
            change_percent=row.get('涨跌幅'),
            change_amount=row.get('涨跌额'),
            turnover_rate=row.get('换手率'),
            pe_ttm=row.get('市盈率(TTM)'),
            pb=row.get('市净率'),
            amplitude=row.get('振幅'),
        )
        session.add(record)

    try:
        session.commit()
        print(f"{symbol} 数据写入完成")
    except IntegrityError:
        session.rollback()
        print(f"{symbol} 部分数据已存在，跳过重复插入")

def main(is_init=False):
    engine = create_engine(DATABASE_URL)
    Session = sessionmaker(bind=engine)
    session = Session()

    # 读所有股票代码
    stock_info_df = pd.read_sql("SELECT symbol FROM stock_basic_info", engine)
    trade_dates = get_trade_dates()
    print(trade_dates)

    if is_init:
        # 初始化拉取最近15个交易日
        last_15_days = trade_dates[-15:]
        start_date = last_15_days[0]
        end_date = last_15_days[-1]
        print(f"初始化拉取日期: {start_date} ~ {end_date}")

        for symbol in stock_info_df['symbol']:
            fetch_and_store_stock_daily(session, symbol, start_date=start_date, end_date=end_date)
    else:
        now = datetime.now()
        if not is_trade_day(now, trade_dates):
            print(f"{now.strftime('%Y-%m-%d')} 非交易日，不更新数据")
            return

        today_str = now.strftime('%Y%m%d')
        for symbol in stock_info_df['symbol']:
            fetch_and_store_stock_daily(session, symbol, start_date=today_str, end_date=today_str)

if __name__ == "__main__":
    # 初始化时传 True，日常更新传 False
    main(is_init=True)
