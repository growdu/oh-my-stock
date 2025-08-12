import configparser
import pandas as pd
import akshare as ak
from sqlalchemy import create_engine, Column, Integer, String, Date, DECIMAL, text
from sqlalchemy.orm import declarative_base, sessionmaker
from datetime import datetime, timedelta 
import time

from sqlalchemy import UniqueConstraint
import os
import requests

os.environ["HTTP_PROXY"] = "http://127.0.0.1:7078"
os.environ["HTTPS_PROXY"] = "http://127.0.0.1:7078"

ak.session = requests.Session()
ak.session.headers.update({
    "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "
                  "AppleWebKit/537.36 (KHTML, like Gecko) "
                  "Chrome/120.0.0.0 Safari/537.36",
    "Accept-Language": "zh-CN,zh;q=0.9"
})

Base = declarative_base()

class StockMoneyFlow(Base):
    __tablename__ = "stock_money_flow"

    id = Column(Integer, primary_key=True)
    symbol = Column(String(10), nullable=False)
    trade_date = Column(Date, nullable=False)
    main_net = Column(DECIMAL(20, 4))
    retail_net = Column(DECIMAL(20, 4))
    large_order_ratio = Column(DECIMAL(10, 4))
    medium_order_ratio = Column(DECIMAL(10, 4))
    small_order_ratio = Column(DECIMAL(10, 4))

    __table_args__ = (
        UniqueConstraint('symbol', 'trade_date', name='uk_stock_money_flow'),
    )

def read_db_url(config_path="config.ini", section="database"):
    config = configparser.ConfigParser()
    config.read(config_path)
    return config[section]['url']

def get_db_engine(db_url):
    return create_engine(db_url)

def get_session(engine):
    return sessionmaker(bind=engine)()

def get_all_symbols(session):
    result = session.execute(text("SELECT symbol FROM stock_basic_info"))
    return [row[0] for row in result]

def get_existing_trade_dates(session, symbol):
    result = session.execute(text("SELECT trade_date FROM stock_money_flow WHERE symbol = :symbol"), {"symbol": symbol})
    return {row[0] for row in result}

def fetch_money_flow(symbol):
    try:
        time.sleep(0.2)  # 避免请求太频繁
        df = ak.stock_individual_fund_flow(symbol)
        if df.empty:
            print(f"{symbol} 无资金流数据")
            return None
        return df
    except Exception as e:
        print(f"获取 {symbol} 资金流数据失败: {e}")
        return None

def main():
    db_url = read_db_url()
    engine = get_db_engine(db_url)
    # Base.metadata.create_all(engine)  # 创建表结构（如果不存在）

    session = get_session(engine)
    symbols = get_all_symbols(session)

    for symbol in symbols:
        print(f"处理 {symbol} ...")
        existing_dates = get_existing_trade_dates(session, symbol)

        df = fetch_money_flow(symbol)
        if df is None:
            continue

        df['trade_date'] = pd.to_datetime(df['日期'], errors='coerce').dt.date
        max_date = df['trade_date'].max()
        min_date = max_date - timedelta(days=7) # 取7天前的数据

        df = df[(df['trade_date'] >= min_date) & (df['trade_date'] <= max_date)]
        recent_dates = sorted(df['trade_date'].unique())[-3:] # 取最近3天的数据
        df = df[df['trade_date'].isin(recent_dates)]

        new_records = []
        for _, row in df.iterrows():
            trade_date = row['trade_date']
            if trade_date in existing_dates:
                print(f"{trade_date} 已导入")
                continue

            record = StockMoneyFlow(
                symbol=symbol,
                trade_date=trade_date,
                main_net=row.get("主力净流入-净额", None),
                retail_net=row.get("小单净流入-净额", None),
                large_order_ratio=row.get("大单净流入-净占比", None),
                medium_order_ratio=row.get("中单净流入-净占比", None),
                small_order_ratio=row.get("小单净流入-净占比", None),
            )
            new_records.append(record)

        if new_records:
            session.add_all(new_records)
            session.commit()
            print(f"{symbol} 插入 {len(new_records)} 条资金流数据")
        else:
            print(f"{symbol} 没有新数据需要插入")

if __name__ == "__main__":
    main()
