export interface StockData {
    code: string; // 股票代码
    name: string; // 股票名称
    currentPrice: number; // 当前价格
    openPrice: number;
    preClose: number;
    high: number;
    low: number;
    volume: number;
    amount: number;
    change: number;
    changePercent: number;
    timestamp: number;
  }
  
  export interface StockState {
    stocks: StockData[];
    watchlist: string[];
    loading: boolean;
    error: string | null;
  }