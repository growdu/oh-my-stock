import akshare as ak
import pandas as pd
import psycopg2
from psycopg2 import sql
from datetime import datetime, timedelta
import time
from tqdm import tqdm
from time import sleep 

# 数据库配置
DB_CONFIG = {
    'host': '192.168.3.99',
    'port': 54322,
    'database': 'stock',
    'user': 'postgres',
    'password': 'double+2=4'
}

# 连接数据库
def get_db_connection():
    return psycopg2.connect(**DB_CONFIG)

# 初始化数据库表
def init_tables():
    conn = get_db_connection()
    cursor = conn.cursor()
    
    # 创建股票基本信息表
    cursor.execute("""
    CREATE TABLE IF NOT EXISTS stock_basic_info (
        code VARCHAR(20) PRIMARY KEY,
        name VARCHAR(50),
        industry VARCHAR(100),
        area VARCHAR(50),
        pe FLOAT,
        outstanding FLOAT,
        totals FLOAT,
        total_assets FLOAT,
        liquid_assets FLOAT,
        fixed_assets FLOAT,
        reserved FLOAT,
        reserved_per_share FLOAT,
        esp FLOAT,
        bvps FLOAT,
        pb FLOAT,
        list_date DATE,
        update_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    )
    """)
    
    # 创建股票历史交易数据表
    cursor.execute("""
    CREATE TABLE IF NOT EXISTS stock_daily_data (
        id SERIAL PRIMARY KEY,
        code VARCHAR(20),
        trade_date DATE,
        open FLOAT,
        high FLOAT,
        low FLOAT,
        close FLOAT,
        volume FLOAT,
        turnover FLOAT,
        amplitude FLOAT,
        change_percent FLOAT,
        change_amount FLOAT,
        turnover_rate FLOAT,
        update_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        UNIQUE (code, trade_date)
    )
    """)
    
    # 创建股票财务报表数据表
    cursor.execute("""
    CREATE TABLE IF NOT EXISTS stock_financial_report (
        id SERIAL PRIMARY KEY,
        code VARCHAR(20),
        report_date DATE,
        eps FLOAT,
        diluted_eps FLOAT,
        total_revenue FLOAT,
        operating_profit FLOAT,
        net_profit FLOAT,
        total_assets FLOAT,
        total_liabilities FLOAT,
        net_assets FLOAT,
        operating_cash_flow FLOAT,
        investing_cash_flow FLOAT,
        financing_cash_flow FLOAT,
        update_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        UNIQUE (code, report_date)
    )
    """)
    
    conn.commit()
    cursor.close()
    conn.close()

# 获取创业板股票列表
def get_cy_stock_list():
    try:
        # 获取A股实时行情数据
        #df = ak.stock_zh_a_spot()
        df = ak.stock_cy_a_spot_em()
        # 筛选创业板股票（代码以300开头）
        cy_stocks = df[df['代码'].str.startswith('300')]
        return cy_stocks[['代码', '名称']].values.tolist()
    except Exception as e:
        print(f"获取创业板股票列表失败: {e}")
        return []

# 更新股票基本信息
def update_stock_basic_info(stock_list):
    conn = get_db_connection()
    cursor = conn.cursor()
    
    for code, name in tqdm(stock_list, desc="更新股票基本信息"):
        try:
            # 获取股票基本信息
            stock_info = ak.stock_individual_info_em(symbol=code)
            if stock_info is None or stock_info.empty:
                continue
                
            # 转换为字典
            info_dict = dict(zip(stock_info['item'], stock_info['value']))
            
            # 准备插入数据
            data = {
                'code': code,
                'name': name,
                'industry': info_dict.get('所属行业', ''),
                'area': info_dict.get('地区', ''),
                'pe': float(info_dict.get('市盈率(动)', 0)),
                'outstanding': float(info_dict.get('流通股(亿)', 0)),
                'totals': float(info_dict.get('总股本(亿)', 0)),
                'total_assets': float(info_dict.get('总资产(亿)', 0)),
                'liquid_assets': float(info_dict.get('流动资产(亿)', 0)),
                'fixed_assets': float(info_dict.get('固定资产(亿)', 0)),
                'reserved': float(info_dict.get('公积金(亿)', 0)),
                'reserved_per_share': float(info_dict.get('每股公积金', 0)),
                'esp': float(info_dict.get('每股收益', 0)),
                'bvps': float(info_dict.get('每股净资产', 0)),
                'pb': float(info_dict.get('市净率', 0)),
                'list_date': datetime.strptime(info_dict.get('上市日期', '2000-01-01'), '%Y-%m-%d').date()
            }
            
            # 插入或更新数据
            insert_sql = """
            INSERT INTO stock_basic_info (
                code, name, industry, area, pe, outstanding, totals, 
                total_assets, liquid_assets, fixed_assets, reserved, 
                reserved_per_share, esp, bvps, pb, list_date
            ) VALUES (
                %(code)s, %(name)s, %(industry)s, %(area)s, %(pe)s, 
                %(outstanding)s, %(totals)s, %(total_assets)s, 
                %(liquid_assets)s, %(fixed_assets)s, %(reserved)s, 
                %(reserved_per_share)s, %(esp)s, %(bvps)s, %(pb)s, %(list_date)s
            )
            ON CONFLICT (code) DO UPDATE SET
                name = EXCLUDED.name,
                industry = EXCLUDED.industry,
                area = EXCLUDED.area,
                pe = EXCLUDED.pe,
                outstanding = EXCLUDED.outstanding,
                totals = EXCLUDED.totals,
                total_assets = EXCLUDED.total_assets,
                liquid_assets = EXCLUDED.liquid_assets,
                fixed_assets = EXCLUDED.fixed_assets,
                reserved = EXCLUDED.reserved,
                reserved_per_share = EXCLUDED.reserved_per_share,
                esp = EXCLUDED.esp,
                bvps = EXCLUDED.bvps,
                pb = EXCLUDED.pb,
                list_date = EXCLUDED.list_date,
                update_time = CURRENT_TIMESTAMP
            """
            
            cursor.execute(insert_sql, data)
            conn.commit()
            
        except Exception as e:
            print(f"更新股票 {code} 基本信息失败: {e}")
            conn.rollback()
            continue
    
    cursor.close()
    conn.close()

# 更新股票历史交易数据
def update_stock_daily_data(stock_list, start_date=None, end_date=None):
    if start_date is None:
        start_date = (datetime.now() - timedelta(days=60)).strftime('%Y%m%d')
    if end_date is None:
        end_date = datetime.now().strftime('%Y%m%d')
    
    conn = get_db_connection()
    cursor = conn.cursor()
    
    for code, name in tqdm(stock_list, desc="更新历史交易数据"):
        try:
            # 获取历史数据
            #hist_data = ak.stock_zh_a_daily(symbol=code, start_date=start_date, end_date=end_date, adjust="qfq")
            hist_data = ak.stock_zh_a_hist(symbol=code, period="daily", adjust="qfq", start_date=start_date, timeout=10)
            if hist_data is None or hist_data.empty:
                continue
                
            # 重置索引并将日期转换为日期对象
            hist_data = hist_data.reset_index()
     
            # 确保日期格式正确
            hist_data['日期'] = pd.to_datetime(hist_data['日期']).dt.date
            
            # 插入数据
            for _, row in hist_data.iterrows():
                insert_sql = """
                INSERT INTO stock_daily_data (
                    code, trade_date, open, high, low, close, 
                    volume, turnover, amplitude, change_percent, 
                    change_amount, turnover_rate
                ) VALUES (
                    %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s
                )
                ON CONFLICT (code, trade_date) DO UPDATE SET
                    open = EXCLUDED.open,
                    high = EXCLUDED.high,
                    low = EXCLUDED.low,
                    close = EXCLUDED.close,
                    volume = EXCLUDED.volume,
                    turnover = EXCLUDED.turnover,
                    amplitude = EXCLUDED.amplitude,
                    change_percent = EXCLUDED.change_percent,
                    change_amount = EXCLUDED.change_amount,
                    turnover_rate = EXCLUDED.turnover_rate,
                    update_time = CURRENT_TIMESTAMP
                """
                
                cursor.execute(insert_sql, (
                    code, 
                    row['日期'], 
                    row['开盘'], 
                    row['最高'], 
                    row['最低'], 
                    row['收盘'], 
                    row['成交量'], 
                    row['成交额'],  # turnover应为成交额（非0）
                    row.get('振幅', 0),  # amplitude需要真实振幅数据
                    row.get('涨跌幅', 0),  # change_percent
                    row.get('涨跌额', 0),  # change_amount
                    row.get('换手率', 0)   # turnover_rate
                ))
            
            conn.commit()
            
        except Exception as e:
            print(f"更新股票 {code} 历史交易数据失败: {e}")
            conn.rollback()
            continue
        
        sleep(2)
    cursor.close()
    conn.close()

# 更新股票财务数据
def update_stock_financial_data(stock_list):
    conn = get_db_connection()
    cursor = conn.cursor()
    
    for code, name in tqdm(stock_list, desc="更新财务数据"):
        try:
            # 获取财务数据
            financial_data = ak.stock_financial_report_sina(stock=code, symbol="现金流量表")
            if financial_data is None or financial_data.empty:
                continue
                
            # 处理财务数据
            for _, row in financial_data.iterrows():
                insert_sql = """
                INSERT INTO stock_financial_report (
                    code, report_date, eps, diluted_eps, total_revenue,
                    operating_profit, net_profit, total_assets,
                    total_liabilities, net_assets, operating_cash_flow,
                    investing_cash_flow, financing_cash_flow
                ) VALUES (
                    %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s
                )
                ON CONFLICT (code, report_date) DO UPDATE SET
                    eps = EXCLUDED.eps,
                    diluted_eps = EXCLUDED.diluted_eps,
                    total_revenue = EXCLUDED.total_revenue,
                    operating_profit = EXCLUDED.operating_profit,
                    net_profit = EXCLUDED.net_profit,
                    total_assets = EXCLUDED.total_assets,
                    total_liabilities = EXCLUDED.total_liabilities,
                    net_assets = EXCLUDED.net_assets,
                    operating_cash_flow = EXCLUDED.operating_cash_flow,
                    investing_cash_flow = EXCLUDED.investing_cash_flow,
                    financing_cash_flow = EXCLUDED.financing_cash_flow,
                    update_time = CURRENT_TIMESTAMP
                """
                
                cursor.execute(insert_sql, (
                    code, row['report_date'], row['eps'], row['diluted_eps'],
                    row['total_revenue'], row['operating_profit'], row['net_profit'],
                    row['total_assets'], row['total_liabilities'], row['net_assets'],
                    row['operating_cash_flow'], row['investing_cash_flow'],
                    row['financing_cash_flow']
                ))
            
            conn.commit()
            
        except Exception as e:
            print(f"更新股票 {code} 财务数据失败: {e}")
            conn.rollback()
            continue
    
    cursor.close()
    conn.close()

# 主函数
def main():
    print("初始化数据库表...")
    #init_tables()
    
    print("获取创业板股票列表...")
    stock_list = get_cy_stock_list()
    # if not stock_list:
    #     print("无法获取创业板股票列表，程序退出")
    #     return
    
    print("更新股票基本信息...")
    #update_stock_basic_info(stock_list)
    
    print("更新股票历史交易数据...")
    update_stock_daily_data(stock_list)
    
    print("更新股票财务数据...")
    update_stock_financial_data(stock_list)
    
    print("数据更新完成")

if __name__ == "__main__":
    main()