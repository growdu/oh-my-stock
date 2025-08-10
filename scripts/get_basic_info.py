import os
from datetime import datetime, timedelta
import pandas as pd
import akshare as ak
from decimal import Decimal
from sqlalchemy import create_engine, Column, String, Date, DECIMAL, BIGINT, Integer, UniqueConstraint, TIMESTAMP
from sqlalchemy.orm import sessionmaker, declarative_base
from sqlalchemy.exc import IntegrityError
import configparser

# ========== 读取配置 ==========
config = configparser.ConfigParser()
config.read("config.ini", encoding="utf-8")
DATABASE_URL = config.get("database", "url")

CACHE_TRADE_DATES_FILE = "cache/trade_dates.csv"
os.makedirs("cache", exist_ok=True)

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
    if os.path.exists(CACHE_TRADE_DATES_FILE):
        df = pd.read_csv(CACHE_TRADE_DATES_FILE, dtype=str)
        return df['trade_date'].tolist()
    else:
        df = ak.trade_date_hist_sina()
        df.to_csv(CACHE_TRADE_DATES_FILE, index=False, encoding='utf-8-sig')
        return df['trade_date'].tolist()

def is_trade_day(date: datetime, trade_dates: list) -> bool:
    return date.strftime("%Y%m%d") in trade_dates

def get_existing_trade_dates(session, symbol):
    rows = session.query(StockDailyData.trade_date).filter(StockDailyData.symbol == symbol).all()
    return set(r[0] for r in rows)

def fetch_and_store_stock_daily(session, symbol, start_date=None, end_date=None):
    if start_date is None:
        start_date = datetime.now().strftime('%Y%m%d')
    if end_date is None:
        end_date = datetime.now().strftime('%Y%m%d')

    try:
        df = ak.stock_zh_a_daily(symbol=symbol, start_date=start_date, end_date=end_date, adjust="qfq")
    except Exception as e:
        print(f"获取{symbol}日线数据失败: {e}")
        return

    if df.empty:
        print(f"{symbol}无日线数据")
        return

    existing_dates = get_existing_trade_dates(session, symbol)

    for _, row in df.iterrows():
        trade_date = row['date']
        if trade_date in existing_dates:
            continue

        record = StockDailyData(
            symbol=symbol,
            trade_date=trade_date,
            open=row.get('open'),
            high=row.get('high'),
            low=row.get('low'),
            close=row.get('close'),
            adj_close=row.get('adj_close'),
            volume=row.get('volume'),
            turnover=row.get('turnover'),
            change_percent=row.get('changepercent'),
            change_amount=row.get('change'),
            turnover_rate=row.get('turnoverrate'),
            pe_ttm=row.get('pe_ttm'),
            pb=row.get('pb'),
            amplitude=row.get('amplitude'),
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

    stock_info_df = pd.read_sql("SELECT symbol FROM stock_basic_info", engine)
    trade_dates = get_trade_dates()

    if is_init:
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
    main(is_init=True)  # 初始化拉取
    # main(is_init=False)  # 日常更新
