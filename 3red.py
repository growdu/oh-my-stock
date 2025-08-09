import akshare as ak
import pandas as pd
import os
from datetime import datetime

# 配置参数
CACHE_FILE = 'cy_board_stocks.csv'  # 本地缓存文件名
CACHE_EXPIRE_DAYS = 7  # 缓存过期天数（1表示1天后需要重新获取）

def get_cy_board_stocks(force_update=False):
    """
    获取创业板股票信息，优先从本地加载，如果本地文件不存在或过期则从网络获取
    
    :param force_update: 是否强制从网络更新数据
    :return: 创业板股票DataFrame
    """
    # 检查是否需要从网络获取数据
    need_update = force_update or not os.path.exists(CACHE_FILE)
    
    if not need_update:
        # 检查缓存是否过期
        file_time = datetime.fromtimestamp(os.path.getmtime(CACHE_FILE))
        if (datetime.now() - file_time).days >= CACHE_EXPIRE_DAYS:
            need_update = True
    
    if need_update:
        print("从网络获取创业板股票数据...")
        try:
            # 获取所有A股实时行情数据
            stock_df = ak.stock_zh_a_spot()
            # 筛选创业板股票（代码以300开头）
            cy_board_stocks = stock_df[stock_df['代码'].str.startswith('300')]
            
            # 保存到本地
            cy_board_stocks.to_csv(CACHE_FILE, index=False, encoding='utf_8_sig')
            print("创业板股票数据已更新并保存到本地")
            return cy_board_stocks
        except Exception as e:
            print(f"从网络获取数据失败: {e}")
            if os.path.exists(CACHE_FILE):
                print("将使用本地缓存数据")
                return pd.read_csv(CACHE_FILE)
            raise
    else:
        print("从本地缓存加载创业板股票数据")
        return pd.read_csv(CACHE_FILE)

def display_stock_info(stock_df):
    """
    显示股票信息
    
    :param stock_df: 股票DataFrame
    """
    print("\n创业板股票列表:")
    print("-" * 80)
    print(f"{'代码':<10}{'名称':<20}{'最新价':<10}{'涨跌幅(%)':<12}{'成交量(手)':<15}{'成交额(万)':<15}")
    print("-" * 80)
    
    for index, row in stock_df.iterrows():
        print(f"{row['代码']:<10}{row['名称']:<20}{row['最新价']:<10.2f}"
              f"{row['涨跌幅']:<12.2f}{row['成交量']:<15}{row['成交额']:<15.2f}")

def main():
    # 获取数据（可以设置force_update=True强制从网络更新）
    try:
        cy_board_stocks = get_cy_board_stocks(force_update=False)
        
        # 显示股票信息
        display_stock_info(cy_board_stocks)
        
        # 这里可以添加其他处理逻辑
        # 例如遍历处理每只股票:
        # for index, row in cy_board_stocks.iterrows():
        #     process_stock(row)
        
    except Exception as e:
        print(f"程序运行出错: {e}")

if __name__ == "__main__":
    main()