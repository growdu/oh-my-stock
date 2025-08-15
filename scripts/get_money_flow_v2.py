# stock_money_flow_importer.py
import akshare as ak
import pandas as pd
import os
from datetime import datetime, timedelta,date
from enum import Enum
import requests
from sqlalchemy import (
    create_engine, Column, Integer, String, Float,
    Date, DateTime, UniqueConstraint
)
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy.orm import sessionmaker
import re
import configparser

# ===== 时间维度枚举 =====
class TimeSpan(Enum):
    REAL_TIME = (0,"即时")
    THREE_TIME= (3,"3日排行")
    FIVE_DAY = (5,"5日排行")
    TEN_DAY = (10, "10日排行")
    
    def __str__(self):
        return self.value[1]

# ===== 缓存机制 =====
CACHE_DIR = "cache"
CACHE_EXPIRE = timedelta(hours=12)

def _get_cache_path(time_span: TimeSpan):
    if not os.path.exists(CACHE_DIR):
        os.makedirs(CACHE_DIR)
    cache_key = f"{time_span.value[1]}_{datetime.now().strftime('%Y%m%d')}"
    filename = str(cache_key)  + ".csv"
    return os.path.join(CACHE_DIR, filename)

def get_cached_data(time_span: TimeSpan):
    cache_file = _get_cache_path(time_span)
    if os.path.exists(cache_file):
        file_time = datetime.fromtimestamp(os.path.getmtime(cache_file))
        if datetime.now() - file_time < CACHE_EXPIRE:
            try:
                df = pd.read_csv(cache_file, quoting=1)
                print(f"从缓存加载 {time_span.value[1]} 数据（{len(df)} 条）")
                return df
            except Exception as e:
                print(f"读取缓存失败: {e}")
    return None

def save_cache(time_span: TimeSpan, df: pd.DataFrame):
    if df.empty:
        return
    cache_file = _get_cache_path(time_span)
    try:
        df.to_csv(cache_file, index=False, encoding="utf-8-sig", quoting=1)
        print(f"已缓存 {time_span.value[1]} 数据到 {cache_file}")
    except Exception as e:
        print(f"缓存保存失败: {e}")

# ===== 数据库导入基类 =====
Base = declarative_base()

class StockMoneyFlow(Base):
    __tablename__ = "stock_money_flow_all"
    id = Column(Integer, primary_key=True, autoincrement=True)
    time_span = Column(Integer, nullable=False)  # 0, 3, 5, 10
    serial_number = Column(Integer)
    symbol = Column(String(10), nullable=False)  # 保留前导零
    name = Column(String(50))
    latest_price = Column(Float)
    change_percent = Column(Float)
    turnover_rate = Column(Float)
    inflow_amount = Column(Float)
    outflow_amount = Column(Float)
    net_amount = Column(Float)
    turnover = Column(Float)
    trade_date = Column(Date, nullable=False)
    created_at = Column(DateTime, default=datetime.now)

    __table_args__ = (
        UniqueConstraint("symbol", "trade_date", "time_span", name="uq_stock_money_flow_all"),
    )
    

# ===== 资金流导入器 =====
def get_money_flow_data(time_span: TimeSpan):
    cached_df = get_cached_data(time_span)
    if cached_df is not None:
        return cached_df
    
    print(f"从 AKShare 获取 {time_span.value[1]} 数据...")
    try:
        df = ak.stock_fund_flow_individual(symbol=time_span.value[1])
        if not df.empty:
                # 确保股票代码列为字符串
            if '股票代码' in df.columns:
                df['股票代码'] = df['股票代码'].astype(str).str.zfill(6) 
            save_cache(time_span, df)
        return df
    except Exception as e:
        print(f"API 请求失败: {e}")
        return pd.DataFrame()

# 工具函数：将带单位的金额转为数字（亿元/万元等）
def parse_amount(amount_str):
    if pd.isna(amount_str):
        return None
    if isinstance(amount_str, (int, float)):
        return float(amount_str)
    amount_str = str(amount_str).strip()
    match = re.match(r"([\d\.]+)([万亿]?)", amount_str)
    if not match:
        return None
    value, unit = match.groups()
    value = float(value)
    if unit == "亿":
        value *= 1e8
    elif unit == "万":
        value *= 1e4
    return value

# 工具函数：百分比转浮点
def parse_percent(percent_str):
    if pd.isna(percent_str):
        return None
    return float(str(percent_str).strip().replace("%", ""))

def read_db_url(config_path="config.ini", section="database"):
    config = configparser.ConfigParser()
    config.read(config_path)
    return config[section]['url']

def get_db_engine(db_url):
    return create_engine(db_url)

def get_session(engine):
    return sessionmaker(bind=engine)()

def import_timespan(session,time_span: TimeSpan):
    # 读取 CSV，保持股票代码前导零
    df = get_money_flow_data(time_span)

    # 解析字段
    records = []
    trade_date = date.today()

    for _, row in df.iterrows():
        record = StockMoneyFlow(
            time_span=time_span.value[0],
            serial_number=int(row["序号"]),
            symbol=row["股票代码"],
            name=row["股票简称"],
            latest_price=float(row["最新价"]),
            change_percent=parse_percent(row["涨跌幅"]),
            turnover_rate=parse_percent(row["换手率"]),
            inflow_amount=parse_amount(row["流入资金"]),
            outflow_amount=parse_amount(row["流出资金"]),
            net_amount=parse_amount(row["净额"]),
            turnover=parse_amount(row["成交额"]),
            trade_date=trade_date,
            created_at=datetime.now()
        )
        records.append(record)
        
    # 入库（去重处理）
    for rec in records:
        exists = session.query(StockMoneyFlow).filter(
            symbol=str(rec.symbol),
            trade_date=rec.trade_date,
            time_span=rec.time_span
        ).first()
        if not exists:
            print(f"{rec.symbol} will import to  money flow")
            session.add(rec)

    session.commit()
    session.close()


def main():
    # 数据库连接
    db_url = read_db_url()
    engine = get_db_engine(db_url)
    Base.metadata.create_all(engine)  # 创建表结构（如果不存在）

    session = get_session(engine)
    for timespan in TimeSpan:
        import_timespan(session, timespan)
        break # 仅仅导入当天的数据
    

ak.session = requests.Session()
ak.session.headers.update({
    "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "
                  "AppleWebKit/537.36 (KHTML, like Gecko) "
                  "Chrome/120.0.0.0 Safari/537.36",
    "Accept-Language": "zh-CN,zh;q=0.9"
})

# ===== 主入口 =====
if __name__ == "__main__":
    print("="*50)
    print("股票资金流数据导入系统")
    print(f"缓存目录: {os.path.abspath(CACHE_DIR)}")
    print(f"缓存有效期: {CACHE_EXPIRE}")
    print("="*50)
    
    main()
    
    print("\n所有数据处理完成")
