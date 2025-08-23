import axios, { AxiosInstance } from "axios";

class SinaStockProvider {
  httpService: AxiosInstance;

  constructor() {
    this.httpService = axios.create({
      timeout: 10000,
      baseURL: "http://192.168.3.99:3003/api/v1",
    });
  }

  /**
   * 拉取股票数据
   * @param codes 股票代码数组，例如 ['sh600000','sz000001']
   */
  async fetch(codes: string[]) {
    const rep = await this.httpService.get("/sina_stocks", {
      params: {
        list: codes.join(","),
      },
    });

    const rawData: string = rep.data;
    // 拆分多只股票的数据
    const rawStocks = rawData
      .split(";")
      .map((v) => v.trim())
      .filter((v) => v.startsWith("var hq_str_"));

    const result = rawStocks.map((line) => {
      // line 格式: var hq_str_sh600000="浦发银行,8.32,8.30,8.35,...";
      const [prefix, data] = line.split("=");
      const code = prefix.replace("var hq_str_", "");
      const fields = data.replace(/"/g, "").split(",");

      return {
        code,
        name: fields[0],
        open: parseFloat(fields[1]),
        yesterdayClose: parseFloat(fields[2]),
        current: parseFloat(fields[3]),
        high: parseFloat(fields[4]),
        low: parseFloat(fields[5]),
        volume: parseFloat(fields[8]), // 成交量
        amount: parseFloat(fields[9]), // 成交额
        date: fields[30],
        time: fields[31],
      };
    });

    return result;
  }
}

export const sinaStockProvider = new SinaStockProvider();
