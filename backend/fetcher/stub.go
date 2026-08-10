package fetcher

import (
	"context"

	"oh-my-stock/models"
)

// 本文件为最小 stub，仅用于让 go build 通过。
// 真实抓取实现由部署侧（容器 / 旧二进制）完成。
// 编译产物只跑 HTTP / 规则 / 预设这些不依赖抓取的功能。

type SinaDaily struct {
	Day      string
	Open     float64
	High     float64
	Low      float64
	Close    float64
	Volume   float64
	Turnover float64
}

type SinaStock struct {
	Code string
	Name string
}

func FetchSinaDaily(_ context.Context, _ string, _ int) ([]SinaDaily, error) {
	return nil, nil
}

func PurgeOldDaily() (int64, error)          { return 0, nil }
func PurgeOldHistoryMV() (int64, error)      { return 0, nil }
func PurgeOldMoneyFlowDaily() (int64, error) { return 0, nil }
func CountBasicInfo() int64                  { return 0 }

func FetchSinaList(_ context.Context) ([]*SinaStock, error) { return nil, nil }
func UpsertBasicInfo(_ []models.StockBasicInfo) (int, error) { return 0, nil }

// UpsertBasicInfoWithValuation 由部署侧提供完整实现；本 stub 仅满足编译。
func UpsertBasicInfoWithValuation(_ []models.StockBasicInfo) (int, error) { return 0, nil }

func UpsertDaily(_ []models.StockDailyData) (int, error) { return 0, nil }
func UpsertHistoryMV(_ []models.StockDailyData) (int, error) { return 0, nil }

func FetchEastMoneyFlowDays(_ context.Context, _ string, _ int) ([]models.StockMoneyFlow, error) {
	return nil, nil
}
func UpsertMoneyFlowDaily(_ []models.StockMoneyFlow) (int, error) { return 0, nil }

func LoadRecentDaily(_ string, _ int) ([]models.StockDailyData, error) { return nil, nil }
func ComputeIndicators(_ string, _ []models.StockDailyData) []models.StockIndicator {
	return nil
}
func UpsertIndicators(_ []models.StockIndicator) (int, error) { return 0, nil }
