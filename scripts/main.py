import akshare as ak
from sqlalchemy import create_engine, Column, Integer, String, Date, Boolean, DECIMAL, TIMESTAMP, text
from sqlalchemy.orm import sessionmaker, declarative_base
from datetime import datetime
import time
import pandas as pd
import requests
import os

Base = declarative_base()

ak.requests = requests.Session()
ak.requests.headers.update({
    "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "
                  "(KHTML, like Gecko) Chrome/115.0 Safari/537.36"
})


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


def parse_decimal(val):
    try:
        return float(val)
    except:
        return None

def is_listed(listing_date_str):
    if listing_date_str:
        return "上市"
    if not listing_date_str:
        return "未知"
    try:
        dt = datetime.strptime(listing_date_str, "%Y-%m-%d")
        if dt > datetime.now():
            return "未上市"
        return "上市"
    except:
        return "未知"

CACHE_DIR = "cache"  # 缓存目录
os.makedirs(CACHE_DIR, exist_ok=True)


def get_or_cache_csv(filename: str, fetch_func, force_update=False):
    """
    优先从本地 CSV 加载数据，如果不存在或 force_update=True，则重新从网络获取并缓存。
    :param filename: 缓存文件名
    :param fetch_func: 数据获取函数（不带参数）
    :param force_update: 是否强制更新缓存
    :return: pandas.DataFrame
    """
    filepath = os.path.join(CACHE_DIR, filename)

    if not force_update and os.path.exists(filepath):
        print(f"从缓存加载: {filepath}")
        return pd.read_csv(filepath)

    print(f"从网络获取: {filename}")
    df = fetch_func()
    df.to_csv(filepath, index=False, encoding="utf-8-sig",quoting=1)
    return df

def normalize_sh_df(df):
    # 上交所数据格式统一
    df = df.rename(columns={
        "证券代码": "代码",
        "证券简称": "简称",
        "公司全称": "公司全称",
        "上市日期": "上市日期"
    })
    # 保留必要列
    return df[["代码", "简称", "公司全称", "上市日期"]]

def normalize_sz_df(df):
    # 深交所数据格式统一
    df = df.rename(columns={
        "A股代码": "代码",
        "A股简称": "简称",
        "A股上市日期": "上市日期"
    })
    df["公司全称"] = None  # 深交所没有公司全称，补空列
    return df[["代码", "简称", "公司全称", "上市日期"]]

def main():
    # 连接 PostgreSQL，改成你自己的连接串
    engine = create_engine("postgresql+psycopg2://postgres:double+2=4@192.168.3.99:54322/a_stock")
    Session = sessionmaker(bind=engine)
    session = Session()

    # 获取股票列表
    # 加载 & 统一格式
    sh_stocks = normalize_sh_df(get_or_cache_csv("sh_stocks.csv", ak.stock_info_sh_name_code))
    sz_stocks = normalize_sz_df(get_or_cache_csv("sz_stocks.csv", ak.stock_info_sz_name_code))

    # 代码转字符串，避免拼接成 float
    sh_stocks["代码"] = sh_stocks["代码"].astype(str).str.strip()
    sz_stocks["代码"] = sz_stocks["代码"].astype(str).str.strip()

    # 拼接
    all_stocks = pd.concat([sh_stocks, sz_stocks], ignore_index=True)

    # 行业映射
    industry_df = get_or_cache_csv("industry.csv", ak.stock_board_industry_name_em)
    industry_map = {}
    industry_map_csv = os.path.join(CACHE_DIR, "industry_map.csv")
    
    # 读取或生成行业映射
    if os.path.exists(industry_map_csv):
        industry_map_df = pd.read_csv(industry_map_csv, dtype={"代码": str})
        industry_map = dict(zip(industry_map_df["代码"], industry_map_df["板块名称"]))
    else:
        for _, row in industry_df.iterrows():
            name = row["板块名称"]
            cons = ak.stock_board_industry_cons_em(name)
            cons["代码"] = cons["代码"].astype(str).str.zfill(6)
            for code in cons["代码"]:
                industry_map[code] = name

        industry_map_df = pd.DataFrame(list(industry_map.items()), columns=["代码", "板块名称"])
        industry_map_df.to_csv(industry_map_csv, index=False, encoding="utf-8-sig",quoting=1)

    for idx, row in all_stocks.iterrows():
        symbol = row['证券代码'] if '证券代码' in row else row.get('代码')
        name = row['证券简称'] if '证券简称' in row else row.get('简称')
        area = row.get('area') or None

        if symbol.startswith('68'):
            market = '科创板'
        elif symbol.startswith('6'):
            market = '主板-沪市'
        elif symbol.startswith('0'):
            market = "主板-深市"
        elif symbol.startswith('3'):
            market = '主板-创业板'
        elif symbol.startswith('8') or symbol.startswith('9'):
            market = '北交所'
        else:
            market = '其他'

        industry = industry_map.get(symbol, None)
        try:
            df = ak.stock_hsgt_individual_em(symbol=symbol)
            if df is None:
                is_hs = False
            else:
                is_hs = not df.empty
        except Exception as e:
            print(f"获取沪深港通持股异常：{e}")
            is_hs = False

        # 获取详细公司信息
        try:
            info = ak.stock_individual_info_em(symbol)
        except Exception as e:
            print(f"获取公司信息失败: {symbol}, {e}")
            info = {}
        info_dict = dict(zip(info['item'], info['value']))
        full_name =  info_dict['股票简称']
        listing_date = info_dict['上市时间']
        status = is_listed(listing_date)
        outstanding_shares = parse_decimal(info_dict['流通股'])
        total_shares = parse_decimal(info_dict['总股本'])

        # 处理 listing_date 字符串转日期
        if listing_date:
            try:
                listing_date = datetime.strptime(str(listing_date), "%Y%m%d").date()
                #listing_date = datetime.strptime(listing_date, "%Y%m%d").strftime("%Y-%m-%d")
            except Exception as e:
                print(f"上市时间异常：{e}")
                listing_date = None
        else:
            listing_date = None

        # 查询是否已存在
        stock_obj = session.query(StockBasicInfo).filter_by(symbol=symbol).first()
        if stock_obj:
            # 更新字段
            stock_obj.name = name
            stock_obj.full_name = full_name
            stock_obj.industry = industry
            stock_obj.area = area
            stock_obj.market = market
            stock_obj.listing_date = listing_date
            stock_obj.outstanding_shares = outstanding_shares
            stock_obj.total_shares = total_shares
            stock_obj.is_hs = is_hs
            stock_obj.status = status
        else:
            stock_obj = StockBasicInfo(
                symbol=symbol,
                name=name,
                full_name=full_name,
                industry=industry,
                area=area,
                market=market,
                listing_date=listing_date,
                outstanding_shares=outstanding_shares,
                total_shares=total_shares,
                is_hs=is_hs,
                status=status
            )
            session.add(stock_obj)

        if idx % 20 == 0:
            session.commit()  # 每20条提交一次，防止内存过大
            print(f"已处理 {idx+1} 条股票数据")
        time.sleep(0.2)  # 防止请求过快被封

    session.commit()
    session.close()
    print("所有股票信息更新完成")

if __name__ == "__main__":
    main()
