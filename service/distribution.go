package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gary0122g/BitfinexFundingData/db"
)

type RateDistribution struct {
	Currency        string    `json:"currency"`
	BinCount        int       `json:"bin_count"`
	MinRate         float64   `json:"min_rate"`
	MaxRate         float64   `json:"max_rate"`
	BinWidth        float64   `json:"bin_width"`
	Distribution    []int     `json:"distribution"`
	PDF             []float64 `json:"pdf"`
	Labels          []string  `json:"labels"`
	TotalTrades     int       `json:"total_trades"`
	LastProcessedID int64     `json:"last_processed_id"`
	LastUpdated     time.Time `json:"last_updated"`
}

type DistributionService struct {
	database *db.Database
}

func NewDistributionService(database *db.Database) *DistributionService {
	return &DistributionService{database: database}
}

// InitializeDistribution 初始化利率分布（處理所有歷史數據）
func (ds *DistributionService) InitializeDistribution(currency string, binCount int) error {
	// 檢查是否已經存在分布
	existing, err := ds.getDistribution(currency, binCount)
	if err == nil && existing != nil {
		fmt.Printf("Distribution already exists for %s with %d bins, %d total trades\n",
			currency, binCount, existing.TotalTrades)
		return nil // 已經存在，不需要重新初始化
	}

	fmt.Printf("No existing distribution found for %s, initializing...\n", currency)

	// 獲取所有交易數據來計算初始分布
	trades, err := ds.database.GetAllWSFundingTrades(currency)
	if err != nil {
		return fmt.Errorf("failed to get trades: %v", err)
	}

	if len(trades) == 0 {
		return fmt.Errorf("no trades found for currency %s", currency)
	}

	// 添加日誌來顯示處理的記錄數量
	fmt.Printf("Initializing distribution for %s with %d trades\n", currency, len(trades))

	// 轉換為 APR 百分比
	rates := make([]float64, len(trades))
	for i, trade := range trades {
		rates[i] = trade.Rate * 365 * 100
	}

	// 計算分布
	distribution := ds.calculateDistribution(rates, binCount)
	distribution.Currency = currency
	distribution.TotalTrades = len(trades)
	if len(trades) > 0 {
		distribution.LastProcessedID = trades[len(trades)-1].ID
	}

	// 保存到資料庫
	return ds.saveDistribution(distribution)
}

// UpdateDistribution 增量更新分布（處理新的交易數據）
func (ds *DistributionService) UpdateDistribution(currency string, binCount int) error {
	// 獲取當前分布
	currentDist, err := ds.getDistribution(currency, binCount)
	if err != nil {
		// 如果沒有現有分布，則初始化
		return ds.InitializeDistribution(currency, binCount)
	}

	// 獲取新的交易數據
	newTrades, err := ds.database.GetWSFundingTradesAfterID(currency, currentDist.LastProcessedID)
	if err != nil {
		return fmt.Errorf("failed to get new trades: %v", err)
	}

	if len(newTrades) == 0 {
		return nil // 沒有新數據
	}

	// 只有當新交易數量達到閾值時才更新（例如10000筆）
	if len(newTrades) < 10000 {
		return nil
	}

	// 更新分布
	for _, trade := range newTrades {
		rate := trade.Rate * 365 * 100
		ds.addRateToDistribution(currentDist, rate)
	}

	currentDist.TotalTrades += len(newTrades)
	currentDist.LastProcessedID = newTrades[len(newTrades)-1].ID
	currentDist.LastUpdated = time.Now()

	// 重新計算PDF
	ds.calculatePDF(currentDist)

	// 保存更新後的分布
	return ds.saveDistribution(currentDist)
}

// calculateDistribution 計算利率分布
func (ds *DistributionService) calculateDistribution(rates []float64, binCount int) *RateDistribution {
	if len(rates) == 0 {
		return nil
	}

	// 找出最大最小值
	minRate := rates[0]
	maxRate := rates[0]
	for _, rate := range rates {
		if rate < minRate {
			minRate = rate
		}
		if rate > maxRate {
			maxRate = rate
		}
	}

	// 擴展範圍以防止邊界問題
	rangeExtension := (maxRate - minRate) * 0.05 // 擴展5%
	minRate -= rangeExtension
	maxRate += rangeExtension

	binWidth := (maxRate - minRate) / float64(binCount)
	if binWidth == 0 {
		binWidth = 1 // 避免除零錯誤
	}

	distribution := &RateDistribution{
		BinCount:     binCount,
		MinRate:      minRate,
		MaxRate:      maxRate,
		BinWidth:     binWidth,
		Distribution: make([]int, binCount),
		Labels:       make([]string, binCount),
		LastUpdated:  time.Now(),
	}

	// 生成標籤
	for i := 0; i < binCount; i++ {
		binStart := minRate + float64(i)*binWidth
		distribution.Labels[i] = fmt.Sprintf("%.2f%%", binStart)
	}

	// 分配數據到箱子中
	for _, rate := range rates {
		ds.addRateToDistribution(distribution, rate)
	}

	// 計算PDF
	ds.calculatePDF(distribution)

	return distribution
}

// addRateToDistribution 將單個利率添加到分布中
func (ds *DistributionService) addRateToDistribution(dist *RateDistribution, rate float64) {
	if rate < dist.MinRate || rate > dist.MaxRate {
		// 如果超出範圍，暫時忽略（在實際使用中可能需要動態擴展範圍）
		return
	}

	binIndex := int((rate - dist.MinRate) / dist.BinWidth)
	if binIndex >= len(dist.Distribution) {
		binIndex = len(dist.Distribution) - 1
	}
	if binIndex < 0 {
		binIndex = 0
	}

	dist.Distribution[binIndex]++
}

// calculatePDF 計算機率密度函數
func (ds *DistributionService) calculatePDF(dist *RateDistribution) {
	total := 0
	for _, count := range dist.Distribution {
		total += count
	}

	dist.PDF = make([]float64, len(dist.Distribution))
	if total > 0 {
		for i, count := range dist.Distribution {
			dist.PDF[i] = float64(count) / float64(total)
		}
	}
}

// saveDistribution 保存分布到資料庫
func (ds *DistributionService) saveDistribution(dist *RateDistribution) error {
	distributionJSON, err := json.Marshal(dist.Distribution)
	if err != nil {
		return err
	}

	query := `
	INSERT OR REPLACE INTO rate_distribution 
	(currency, bin_count, min_rate, max_rate, bin_width, distribution, total_trades, last_processed_trade_id, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = ds.database.GetDB().Exec(query,
		dist.Currency,
		dist.BinCount,
		dist.MinRate,
		dist.MaxRate,
		dist.BinWidth,
		string(distributionJSON),
		dist.TotalTrades,
		dist.LastProcessedID,
		time.Now().UnixMilli())

	return err
}

// getDistribution 從資料庫獲取分布
func (ds *DistributionService) getDistribution(currency string, binCount int) (*RateDistribution, error) {
	query := `
	SELECT min_rate, max_rate, bin_width, distribution, total_trades, last_processed_trade_id, updated_at
	FROM rate_distribution 
	WHERE currency = ? AND bin_count = ?`

	var distributionJSON string
	var updatedAt int64
	dist := &RateDistribution{
		Currency: currency,
		BinCount: binCount,
	}

	err := ds.database.GetDB().QueryRow(query, currency, binCount).Scan(
		&dist.MinRate,
		&dist.MaxRate,
		&dist.BinWidth,
		&distributionJSON,
		&dist.TotalTrades,
		&dist.LastProcessedID,
		&updatedAt)

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(distributionJSON), &dist.Distribution)
	if err != nil {
		return nil, err
	}

	dist.LastUpdated = time.Unix(updatedAt/1000, 0)

	// 生成標籤和PDF
	dist.Labels = make([]string, binCount)
	for i := 0; i < binCount; i++ {
		binStart := dist.MinRate + float64(i)*dist.BinWidth
		dist.Labels[i] = fmt.Sprintf("%.2f%%", binStart)
	}

	ds.calculatePDF(dist)

	return dist, nil
}

// GetDistribution 公開方法獲取分布，如果不存在則自動初始化
func (ds *DistributionService) GetDistribution(currency string, binCount int) (*RateDistribution, error) {
	// 先嘗試獲取現有分布
	dist, err := ds.getDistribution(currency, binCount)
	if err == nil {
		return dist, nil
	}

	// 如果不存在，則初始化
	err = ds.InitializeDistribution(currency, binCount)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize distribution: %v", err)
	}

	// 再次獲取
	return ds.getDistribution(currency, binCount)
}
