package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"investorcenter-api/database"
	"investorcenter-api/models"
	"investorcenter-api/services"

	"github.com/gin-gonic/gin"
)

// FundamentalsHandler handles all fundamentals enhancement endpoints
type FundamentalsHandler struct{}

// NewFundamentalsHandler creates a new FundamentalsHandler
func NewFundamentalsHandler() *FundamentalsHandler {
	return &FundamentalsHandler{}
}

// ============================================================================
// GetSectorPercentiles — GET /stocks/:ticker/sector-percentiles
// ============================================================================

func (h *FundamentalsHandler) GetSectorPercentiles(c *gin.Context) {
	ticker := strings.ToUpper(c.Param("ticker"))
	if ticker == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ticker symbol is required"})
		return
	}

	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Database not available",
			"message": "Sector percentiles are temporarily unavailable",
		})
		return
	}

	// Get stock's sector
	stock, err := database.GetStockBySymbol(ticker)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Stock not found",
			"message": fmt.Sprintf("No data available for %s", ticker),
			"ticker":  ticker,
		})
		return
	}

	if stock.Sector == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Sector not available",
			"message": fmt.Sprintf("No sector classification available for %s", ticker),
			"ticker":  ticker,
		})
		return
	}

	// Fetch sector percentiles and stock metrics in parallel
	var (
		percentiles []models.SectorPercentile
		metricsMap  map[string]*float64
		percErr     error
		metricsErr  error
	)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		percentiles, percErr = database.GetSectorPercentiles(stock.Sector)
	}()
	go func() {
		defer wg.Done()
		metricsMap, _, metricsErr = database.GetStockMetricsMap(ticker)
	}()
	wg.Wait()

	if percErr != nil {
		log.Printf("Error fetching sector percentiles for %s: %v", stock.Sector, percErr)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch sector percentiles",
			"message": "An error occurred while retrieving sector data",
		})
		return
	}

	if len(percentiles) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "No percentile data",
			"message": fmt.Sprintf("No sector percentile data available for sector %s", stock.Sector),
			"ticker":  ticker,
		})
		return
	}

	if metricsErr != nil {
		log.Printf("Warning: failed to get stock metrics for %s: %v", ticker, metricsErr)
	}

	// Parse optional metrics filter
	metricsFilter := make(map[string]bool)
	if filterParam := c.Query("metrics"); filterParam != "" {
		for _, m := range strings.Split(filterParam, ",") {
			metricsFilter[strings.TrimSpace(m)] = true
		}
	}

	// Build response
	var sampleCount *int
	metricsResponse := make(map[string]*models.MetricPercentileData)

	for _, sp := range percentiles {
		// Apply filter if provided
		if len(metricsFilter) > 0 && !metricsFilter[sp.MetricName] {
			continue
		}

		if sampleCount == nil {
			sampleCount = sp.SampleCount
		}

		data := &models.MetricPercentileData{
			LowerIsBetter: models.LowerIsBetterMetrics[sp.MetricName],
			SampleCount:   sp.SampleCount,
		}

		// Build distribution
		data.Distribution = buildDistribution(&sp)

		// Calculate stock's percentile if we have its metric value
		if metricsMap != nil {
			if val, ok := metricsMap[sp.MetricName]; ok && val != nil {
				data.Value = val
				pct := percentileFromDistribution(&sp, *val)
				data.Percentile = &pct
			}
		}

		metricsResponse[sp.MetricName] = data
	}

	calculatedAt := ""
	if len(percentiles) > 0 {
		calculatedAt = percentiles[0].CalculatedAt.Format("2006-01-02")
	}

	c.JSON(http.StatusOK, gin.H{
		"data": models.SectorPercentilesResponse{
			Ticker:       ticker,
			Sector:       stock.Sector,
			CalculatedAt: calculatedAt,
			SampleCount:  sampleCount,
			Metrics:      metricsResponse,
		},
		"meta": gin.H{
			"source":       "mv_latest_sector_percentiles",
			"metric_count": len(metricsResponse),
			"timestamp":    time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// ============================================================================
// GetStockPeers — GET /stocks/:ticker/peers
// ============================================================================

func (h *FundamentalsHandler) GetStockPeers(c *gin.Context) {
	ticker := strings.ToUpper(c.Param("ticker"))
	if ticker == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ticker symbol is required"})
		return
	}

	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Database not available",
			"message": "Peer comparison is temporarily unavailable",
		})
		return
	}

	limit := 5
	if limitStr := c.DefaultQuery("limit", "5"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 10 {
			limit = parsed
		}
	}

	// Get stock info
	stock, err := database.GetStockBySymbol(ticker)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Stock not found",
			"message": fmt.Sprintf("No data available for %s", ticker),
			"ticker":  ticker,
		})
		return
	}

	var marketCap float64
	if stock.MarketCap != nil {
		marketCap, _ = stock.MarketCap.Float64()
	}

	if marketCap == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Market cap not available",
			"message": fmt.Sprintf("Market cap data not available for %s, cannot determine peers", ticker),
			"ticker":  ticker,
		})
		return
	}

	// Try industry-level peers first, fall back to sector
	peerSource := "industry"
	var peers []models.EnrichedPeer

	if stock.Industry != "" {
		peers, err = database.GetEnrichedIndustryPeers(stock.Industry, marketCap, ticker, limit)
		if err != nil {
			log.Printf("Error fetching industry peers for %s: %v", ticker, err)
		}
	}

	if len(peers) < 3 && stock.Sector != "" {
		peerSource = "sector"
		peers, err = database.GetEnrichedSectorPeers(stock.Sector, marketCap, ticker, limit)
		if err != nil {
			log.Printf("Error fetching sector peers for %s: %v", ticker, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to fetch peers",
				"message": "An error occurred while retrieving peer data",
			})
			return
		}
	}

	// Get stock's own IC Score and metrics
	var stockICScore *float64
	var stockMetrics *models.PeerMetrics

	var metricsRow *models.StockMetricsRow
	var icScore *models.ICScore

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		icScore, _ = database.GetLatestICScore(ticker)
	}()
	go func() {
		defer wg.Done()
		_, metricsRow, _ = database.GetStockMetricsMap(ticker)
	}()
	wg.Wait()

	if icScore != nil {
		f, _ := icScore.OverallScore.Float64()
		stockICScore = &f
	}

	if metricsRow != nil {
		stockMetrics = &models.PeerMetrics{
			PERatio:          metricsRow.PERatio,
			ROE:              metricsRow.ROE,
			RevenueGrowthYoY: metricsRow.RevenueGrowthYoY,
			NetMargin:        metricsRow.NetMargin,
			DebtToEquity:     metricsRow.DebtToEquity,
			MarketCap:        &marketCap,
		}
	}

	// Build peer response
	peerData := make([]models.PeerData, 0, len(peers))
	var totalScore float64
	var scoreCount int

	for _, p := range peers {
		pd := models.PeerData{
			Ticker:      p.Symbol,
			CompanyName: p.Name,
			ICScore:     p.ICScore,
			Industry:    p.Industry,
			Metrics: &models.PeerMetrics{
				PERatio:          p.PERatio,
				ROE:              p.ROE,
				RevenueGrowthYoY: p.RevenueGrowthYoY,
				NetMargin:        p.NetMargin,
				DebtToEquity:     p.DebtToEquity,
				MarketCap:        p.MarketCap,
			},
		}
		peerData = append(peerData, pd)

		if p.ICScore != nil {
			totalScore += *p.ICScore
			scoreCount++
		}
	}

	response := models.PeersResponse{
		Ticker:       ticker,
		ICScore:      stockICScore,
		Industry:     stock.Industry,
		Peers:        peerData,
		StockMetrics: stockMetrics,
	}

	if scoreCount > 0 {
		avg := totalScore / float64(scoreCount)
		response.AvgPeerScore = &avg
		if stockICScore != nil {
			delta := *stockICScore - avg
			response.VsPeersDelta = &delta
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": response,
		"meta": gin.H{
			"peer_selection": fmt.Sprintf("%s + market cap proximity", peerSource),
			"peer_count":     len(peerData),
			"timestamp":      time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// ============================================================================
// GetFairValue — GET /stocks/:ticker/fair-value
// ============================================================================

func (h *FundamentalsHandler) GetFairValue(c *gin.Context) {
	ticker := strings.ToUpper(c.Param("ticker"))
	if ticker == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ticker symbol is required"})
		return
	}

	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Database not available",
			"message": "Fair value estimates are temporarily unavailable",
		})
		return
	}

	// Fetch fair value metrics from DB and FMP data in parallel
	var (
		dbFairValue   *models.FairValueMetrics
		fmpRatios     *FMPRatiosTTMWrapper
		priceTarget   *PriceTargetWrapper
		dbErr, fmpErr error
		ptErr         error
	)

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		dbFairValue, dbErr = database.GetFairValueMetrics(ticker)
	}()
	go func() {
		defer wg.Done()
		if fmpClient != nil {
			ratios, err := fmpClient.GetRatiosTTM(ticker)
			if err != nil {
				fmpErr = err
			} else {
				fmpRatios = &FMPRatiosTTMWrapper{ratios}
			}
		}
	}()
	go func() {
		defer wg.Done()
		if fmpClient != nil {
			pt, err := fmpClient.GetPriceTargetConsensus(ticker)
			if err != nil {
				ptErr = err
			} else {
				priceTarget = &PriceTargetWrapper{pt}
			}
		}
	}()
	wg.Wait()

	if dbErr != nil {
		log.Printf("Error fetching fair value metrics for %s: %v", ticker, dbErr)
	}
	if fmpErr != nil {
		log.Printf("Warning: FMP ratios unavailable for %s: %v", ticker, fmpErr)
	}
	if ptErr != nil {
		log.Printf("Warning: FMP price target unavailable for %s: %v", ticker, ptErr)
	}

	// Determine current price
	var currentPrice *float64
	if dbFairValue != nil && dbFairValue.StockPrice != nil {
		currentPrice = dbFairValue.StockPrice
	}

	// Build fair value models
	fairValueModels := make(map[string]*models.FairValueModel)

	// DCF model
	if dbFairValue != nil && dbFairValue.DCFFairValue != nil {
		dcf := &models.FairValueModel{
			FairValue:  dbFairValue.DCFFairValue,
			Confidence: "medium",
		}
		if currentPrice != nil && *currentPrice > 0 {
			upside := (*dbFairValue.DCFFairValue - *currentPrice) / *currentPrice * 100
			dcf.UpsidePercent = &upside
		}
		if dbFairValue.WACC != nil {
			dcf.Inputs = map[string]interface{}{
				"wacc": *dbFairValue.WACC,
			}
		}
		fairValueModels["dcf"] = dcf
	}

	// Graham Number
	grahamValue := getGrahamNumber(dbFairValue)
	if grahamValue == nil && fmpRatios != nil {
		grahamValue = fmpRatios.GrahamNumberTTM
	}
	if grahamValue != nil {
		graham := &models.FairValueModel{
			FairValue:  grahamValue,
			Confidence: "high",
		}
		if currentPrice != nil && *currentPrice > 0 {
			upside := (*grahamValue - *currentPrice) / *currentPrice * 100
			graham.UpsidePercent = &upside
		}
		fairValueModels["graham_number"] = graham
	}

	// EPV model
	if dbFairValue != nil && dbFairValue.EPVFairValue != nil {
		epv := &models.FairValueModel{
			FairValue:  dbFairValue.EPVFairValue,
			Confidence: "medium",
		}
		if currentPrice != nil && *currentPrice > 0 {
			upside := (*dbFairValue.EPVFairValue - *currentPrice) / *currentPrice * 100
			epv.UpsidePercent = &upside
		}
		fairValueModels["epv"] = epv
	}

	// Suppression check: suppress if no models could be computed
	suppressed := len(fairValueModels) == 0
	var suppressionReason *string
	if suppressed {
		reason := "Insufficient financial data to compute fair value estimates"
		suppressionReason = &reason
	}

	// Analyst consensus
	var analystConsensus *models.FVAnalystConsensus
	if priceTarget != nil && priceTarget.TargetConsensus != nil {
		ac := &models.FVAnalystConsensus{
			TargetPrice: priceTarget.TargetConsensus,
		}
		if currentPrice != nil && *currentPrice > 0 {
			upside := (*priceTarget.TargetConsensus - *currentPrice) / *currentPrice * 100
			ac.UpsidePercent = &upside
		}
		analystConsensus = ac
	}

	// Margin of safety
	mos := computeMarginOfSafety(fairValueModels, currentPrice)

	c.JSON(http.StatusOK, gin.H{
		"data": models.FairValueResponse{
			Ticker:             ticker,
			CurrentPrice:       currentPrice,
			Models:             fairValueModels,
			FVAnalystConsensus: analystConsensus,
			MarginOfSafety:     mos,
			Suppressed:         suppressed,
			SuppressionReason:  suppressionReason,
		},
		"meta": gin.H{
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// ============================================================================
// GetHealthSummary — GET /stocks/:ticker/health-summary
// ============================================================================

func (h *FundamentalsHandler) GetHealthSummary(c *gin.Context) {
	ticker := strings.ToUpper(c.Param("ticker"))
	if ticker == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ticker symbol is required"})
		return
	}

	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Database not available",
			"message": "Health summary is temporarily unavailable",
		})
		return
	}

	// Get stock's sector
	stock, err := database.GetStockBySymbol(ticker)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Stock not found",
			"message": fmt.Sprintf("No data available for %s", ticker),
			"ticker":  ticker,
		})
		return
	}

	// Fetch all data sources in parallel
	var (
		icScore     *models.ICScore
		fmpScore    *FMPScoreWrapper
		lifecycle   *models.LifecycleClassification
		metricsMap  map[string]*float64
		metricsRow  *models.StockMetricsRow
		percentiles []models.SectorPercentile

		icErr, fmpScoreErr, lcErr, metricsErr, percErr error
	)

	var wg sync.WaitGroup
	wg.Add(5)
	go func() {
		defer wg.Done()
		icScore, icErr = database.GetLatestICScore(ticker)
	}()
	go func() {
		defer wg.Done()
		if fmpClient != nil {
			score, err := fmpClient.GetScore(ticker)
			if err != nil {
				fmpScoreErr = err
			} else {
				fmpScore = &FMPScoreWrapper{score}
			}
		}
	}()
	go func() {
		defer wg.Done()
		lifecycle, lcErr = database.GetLifecycleClassification(ticker)
	}()
	go func() {
		defer wg.Done()
		metricsMap, metricsRow, metricsErr = database.GetStockMetricsMap(ticker)
	}()
	go func() {
		defer wg.Done()
		if stock.Sector != "" {
			percentiles, percErr = database.GetSectorPercentiles(stock.Sector)
		}
	}()
	wg.Wait()

	// Log non-critical errors
	if icErr != nil {
		log.Printf("Warning: IC Score unavailable for %s: %v", ticker, icErr)
	}
	if fmpScoreErr != nil {
		log.Printf("Warning: FMP Score unavailable for %s: %v", ticker, fmpScoreErr)
	}
	if lcErr != nil {
		log.Printf("Warning: Lifecycle classification unavailable for %s: %v", ticker, lcErr)
	}
	if metricsErr != nil {
		log.Printf("Warning: Stock metrics unavailable for %s: %v", ticker, metricsErr)
	}
	if percErr != nil {
		log.Printf("Warning: Sector percentiles unavailable for %s: %v", ticker, percErr)
	}

	// Check if we have sufficient data to produce a meaningful response
	dataQuality := "full"
	sourcesAvailable := 0
	if icScore != nil {
		sourcesAvailable++
	}
	if fmpScore != nil {
		sourcesAvailable++
	}
	if lifecycle != nil {
		sourcesAvailable++
	}
	if metricsMap != nil {
		sourcesAvailable++
	}
	if len(percentiles) > 0 {
		sourcesAvailable++
	}
	switch {
	case sourcesAvailable == 0:
		dataQuality = "insufficient"
	case sourcesAvailable <= 2:
		dataQuality = "partial"
	}

	// Compute percentiles for each metric
	percentileMap := make(map[string]*float64)
	if metricsMap != nil && len(percentiles) > 0 {
		spMap := make(map[string]*models.SectorPercentile)
		for i := range percentiles {
			spMap[percentiles[i].MetricName] = &percentiles[i]
		}
		for name, val := range metricsMap {
			if val == nil {
				continue
			}
			if sp, ok := spMap[name]; ok {
				pct := percentileFromDistribution(sp, *val)
				percentileMap[name] = &pct
			}
		}
	}

	// Build health badge
	var fScorePtr *int
	var zScorePtr *float64
	var icHealthPtr *float64

	if fmpScore != nil && fmpScore.PiotroskiScore != nil {
		fScorePtr = fmpScore.PiotroskiScore
	}
	if fmpScore != nil && fmpScore.AltmanZScore != nil {
		zScorePtr = fmpScore.AltmanZScore
	}
	if icScore != nil && icScore.FinancialHealthScore != nil {
		f, _ := icScore.FinancialHealthScore.Float64()
		icHealthPtr = &f
	}

	badge, score := computeHealthBadge(fScorePtr, zScorePtr, icHealthPtr, percentileMap["debt_to_equity"])

	// Build health components
	components := make(map[string]*models.HealthComponent)
	if fScorePtr != nil {
		interp := "Moderate"
		if *fScorePtr >= 7 {
			interp = "Strong"
		} else if *fScorePtr <= 3 {
			interp = "Weak"
		}
		components["piotroski_f_score"] = &models.HealthComponent{
			Value: float64(*fScorePtr), Max: floatPtr(9), Interpretation: interp,
		}
	}
	if zScorePtr != nil {
		zone := "grey"
		interp := "Grey zone"
		if *zScorePtr > 2.99 {
			zone = "safe"
			interp = "Healthy"
		} else if *zScorePtr < 1.81 {
			zone = "distress"
			interp = "Distress zone"
		}
		components["altman_z_score"] = &models.HealthComponent{
			Value: *zScorePtr, Zone: zone, Interpretation: interp,
		}

	}
	if icHealthPtr != nil {
		components["ic_financial_health"] = &models.HealthComponent{
			Value: *icHealthPtr, Max: floatPtr(100), Interpretation: fmt.Sprintf("%.0f/100", *icHealthPtr),
		}
	}
	if dePct, ok := percentileMap["debt_to_equity"]; ok && dePct != nil {
		interp := "Moderate leverage"
		if *dePct >= 80 {
			interp = "Low leverage"
		} else if *dePct <= 20 {
			interp = "High leverage"
		}
		components["debt_percentile"] = &models.HealthComponent{
			Value: math.Round(*dePct), Interpretation: interp,
		}
	}

	// Build lifecycle info
	var lifecycleInfo *models.LifecycleInfo
	if lifecycle != nil {
		lifecycleInfo = &models.LifecycleInfo{
			Stage:        lifecycle.LifecycleStage,
			Description:  models.GetLifecycleDescription(lifecycle.LifecycleStage),
			ClassifiedAt: lifecycle.ClassifiedAt.Format("2006-01-02"),
		}
	}

	// Generate strengths and concerns
	strengths := generateStrengths(metricsMap, percentileMap, stock.Sector, 3)
	concerns := generateConcerns(metricsMap, percentileMap, stock.Sector, 3)

	// Detect red flags
	redFlags := detectRedFlags(metricsRow, fScorePtr, zScorePtr, percentileMap)

	c.JSON(http.StatusOK, gin.H{
		"data": models.HealthSummaryResponse{
			Ticker: ticker,
			Health: &models.HealthBadge{
				Badge:      badge,
				Score:      math.Round(score*10) / 10,
				Components: components,
			},
			Lifecycle: lifecycleInfo,
			Strengths: strengths,
			Concerns:  concerns,
			RedFlags:  redFlags,
		},
		"meta": gin.H{
			"data_quality":      dataQuality,
			"sources_available": sourcesAvailable,
			"timestamp":         time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// ============================================================================
// GetMetricHistory — GET /stocks/:ticker/metric-history/:metric
// ============================================================================

func (h *FundamentalsHandler) GetMetricHistory(c *gin.Context) {
	ticker := strings.ToUpper(c.Param("ticker"))
	metric := strings.ToLower(c.Param("metric"))

	if ticker == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ticker symbol is required"})
		return
	}

	// Validate metric name against known mappings
	mapping, ok := models.MetricStatementMap[metric]
	if !ok {
		validMetrics := make([]string, 0, len(models.MetricStatementMap))
		for k := range models.MetricStatementMap {
			validMetrics = append(validMetrics, k)
		}
		sort.Strings(validMetrics)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         "Unknown metric",
			"message":       fmt.Sprintf("Metric '%s' is not supported", metric),
			"valid_metrics": validMetrics,
		})
		return
	}

	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Database not available",
			"message": "Metric history is temporarily unavailable",
		})
		return
	}

	timeframe := c.DefaultQuery("timeframe", "quarterly")
	if timeframe != "quarterly" && timeframe != "annual" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid timeframe",
			"message": "Timeframe must be 'quarterly' or 'annual'",
		})
		return
	}

	limit := 20
	if limitStr := c.DefaultQuery("limit", "20"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 40 {
			limit = parsed
		}
	}

	// Fetch metric history
	rows, err := database.GetMetricHistory(ticker, mapping.StatementType, mapping.FieldName, timeframe, limit)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "No data found",
				"message": fmt.Sprintf("No %s history available for %s", metric, ticker),
				"ticker":  ticker,
			})
			return
		}
		log.Printf("Error fetching metric history for %s/%s: %v", ticker, metric, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch metric history",
			"message": "An error occurred while retrieving historical data",
		})
		return
	}

	if len(rows) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "No data found",
			"message": fmt.Sprintf("No %s history available for %s", metric, ticker),
			"ticker":  ticker,
		})
		return
	}

	// Convert to response format and compute YoY changes
	dataPoints := make([]models.MetricDataPoint, len(rows))
	for i, row := range rows {
		dataPoints[i] = models.MetricDataPoint{
			PeriodEnd:     row.PeriodEnd,
			FiscalYear:    row.FiscalYear,
			FiscalQuarter: row.FiscalQuarter,
			Value:         row.Value,
		}
	}

	// Compute YoY changes
	lookback := 4 // quarterly: 4 periods back = 1 year
	if timeframe == "annual" {
		lookback = 1
	}
	for i := range dataPoints {
		if i+lookback < len(dataPoints) {
			cur := dataPoints[i].Value
			prev := dataPoints[i+lookback].Value
			if cur != nil && prev != nil && math.Abs(*prev) > 1e-10 {
				change := (*cur - *prev) / math.Abs(*prev)
				dataPoints[i].YoYChange = &change
			}
		}
	}

	// Compute trend
	trend := computeTrend(dataPoints, timeframe)

	c.JSON(http.StatusOK, gin.H{
		"data": models.MetricHistoryResponse{
			Ticker:     ticker,
			Metric:     metric,
			Timeframe:  timeframe,
			Unit:       mapping.Unit,
			DataPoints: dataPoints,
			Trend:      trend,
		},
		"meta": gin.H{
			"available_periods": len(dataPoints),
			"source":            "financial_statements",
			"timestamp":         time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// ============================================================================
// Helper Types (thin wrappers to avoid nil pointer issues with FMP types)
// ============================================================================

type FMPRatiosTTMWrapper struct {
	*services.FMPRatiosTTM
}

type FMPScoreWrapper struct {
	*services.FMPScore
}

type PriceTargetWrapper struct {
	*services.FMPPriceTargetConsensus
}

// getGrahamNumber safely extracts graham number from FairValueMetrics
func getGrahamNumber(fv *models.FairValueMetrics) *float64 {
	if fv == nil {
		return nil
	}
	return fv.GrahamNumber
}

// ============================================================================
// Health Badge Computation (deterministic algorithm from tech spec)
// ============================================================================

func computeHealthBadge(fScore *int, zScore *float64, icHealth *float64, dePercentile *float64) (string, float64) {
	score := 0.0

	// F-Score contribution (0-9 → 0-30 points)
	if fScore != nil {
		score += float64(*fScore) / 9.0 * 30.0
	}

	// Z-Score contribution (0-30 points)
	if zScore != nil {
		switch {
		case *zScore > 2.99:
			score += 30.0
		case *zScore > 1.81:
			score += 15.0
		}
	}

	// IC Health contribution (0-100 → 0-25 points)
	if icHealth != nil {
		score += *icHealth / 100.0 * 25.0
	}

	// D/E percentile contribution (inverted, 0-15 points)
	// Note: percentile is already direction-adjusted (higher = better)
	if dePercentile != nil {
		score += *dePercentile / 100.0 * 15.0
	}

	badge := "Distressed"
	switch {
	case score >= 80:
		badge = "Strong"
	case score >= 65:
		badge = "Healthy"
	case score >= 45:
		badge = "Fair"
	case score >= 25:
		badge = "Weak"
	}

	return badge, score
}

// ============================================================================
// Margin of Safety Computation
// ============================================================================

func computeMarginOfSafety(fairValueModels map[string]*models.FairValueModel, currentPrice *float64) *models.MarginOfSafety {
	if currentPrice == nil || *currentPrice == 0 || len(fairValueModels) == 0 {
		return nil
	}

	var totalFV float64
	count := 0
	for _, m := range fairValueModels {
		if m.FairValue != nil {
			totalFV += *m.FairValue
			count++
		}
	}
	if count == 0 {
		return nil
	}

	avgFV := totalFV / float64(count)
	diff := (avgFV - *currentPrice) / *currentPrice * 100

	var zone, description string
	switch {
	case diff > 15:
		zone = "undervalued"
		description = fmt.Sprintf("Stock may be undervalued — trading %.1f%% below average fair value estimate", diff)
	case diff < -15:
		zone = "overvalued"
		description = fmt.Sprintf("Stock may be overvalued — trading %.1f%% above average fair value estimate", -diff)
	default:
		zone = "fairly_valued"
		description = "Stock is trading within 15% of average fair value estimate"
	}

	return &models.MarginOfSafety{
		AvgFairValue: &avgFV,
		Zone:         zone,
		Description:  description,
	}
}

// ============================================================================
// Percentile Calculation (in-handler, avoids extra DB calls)
// ============================================================================

func buildDistribution(sp *models.SectorPercentile) *models.PercentileDistribution {
	d := &models.PercentileDistribution{}
	if sp.MinValue != nil {
		d.Min, _ = sp.MinValue.Float64()
	}
	if sp.P10Value != nil {
		d.P10, _ = sp.P10Value.Float64()
	}
	if sp.P25Value != nil {
		d.P25, _ = sp.P25Value.Float64()
	}
	if sp.P50Value != nil {
		d.P50, _ = sp.P50Value.Float64()
	}
	if sp.P75Value != nil {
		d.P75, _ = sp.P75Value.Float64()
	}
	if sp.P90Value != nil {
		d.P90, _ = sp.P90Value.Float64()
	}
	if sp.MaxValue != nil {
		d.Max, _ = sp.MaxValue.Float64()
	}
	return d
}

func percentileFromDistribution(sp *models.SectorPercentile, value float64) float64 {
	d := buildDistribution(sp)

	var rawPct float64
	switch {
	case value <= d.Min:
		rawPct = 0
	case value >= d.Max:
		rawPct = 100
	case value <= d.P10:
		rawPct = interpolateFund(value, d.Min, d.P10, 0, 10)
	case value <= d.P25:
		rawPct = interpolateFund(value, d.P10, d.P25, 10, 25)
	case value <= d.P50:
		rawPct = interpolateFund(value, d.P25, d.P50, 25, 50)
	case value <= d.P75:
		rawPct = interpolateFund(value, d.P50, d.P75, 50, 75)
	case value <= d.P90:
		rawPct = interpolateFund(value, d.P75, d.P90, 75, 90)
	default:
		rawPct = interpolateFund(value, d.P90, d.Max, 90, 100)
	}

	// Invert for "lower is better" metrics
	if models.LowerIsBetterMetrics[sp.MetricName] {
		rawPct = 100 - rawPct
	}

	return math.Round(rawPct*10) / 10
}

func interpolateFund(value, lowVal, highVal, lowPct, highPct float64) float64 {
	if highVal == lowVal {
		return lowPct
	}
	ratio := (value - lowVal) / (highVal - lowVal)
	return lowPct + ratio*(highPct-lowPct)
}

// ============================================================================
// Strengths & Concerns Generation
// ============================================================================

// metricDisplayNames maps metric keys to human-readable names
var metricDisplayNames = map[string]string{
	"gross_margin":       "Gross margin",
	"operating_margin":   "Operating margin",
	"net_margin":         "Net margin",
	"ebitda_margin":      "EBITDA margin",
	"roe":                "Return on equity",
	"roa":                "Return on assets",
	"roic":               "Return on invested capital",
	"revenue_growth_yoy": "Revenue growth (YoY)",
	"eps_growth_yoy":     "EPS growth (YoY)",
	"current_ratio":      "Current ratio",
	"quick_ratio":        "Quick ratio",
	"debt_to_equity":     "Debt/Equity",
	"interest_coverage":  "Interest coverage",
	"dividend_yield":     "Dividend yield",
	"pe_ratio":           "P/E ratio",
	"pb_ratio":           "P/B ratio",
	"ps_ratio":           "P/S ratio",
	"ev_to_ebitda":       "EV/EBITDA",
}

func generateStrengths(metricsMap map[string]*float64, percentileMap map[string]*float64, sector string, limit int) []models.StrengthConcern {
	type metricPct struct {
		name string
		pct  float64
		val  float64
	}

	var candidates []metricPct
	for name, pct := range percentileMap {
		if pct == nil || *pct < 75 {
			continue
		}
		val := metricsMap[name]
		v := 0.0
		if val != nil {
			v = *val
		}
		candidates = append(candidates, metricPct{name: name, pct: *pct, val: v})
	}

	// Sort by percentile descending
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].pct > candidates[j].pct
	})

	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	strengths := make([]models.StrengthConcern, len(candidates))
	for i, c := range candidates {
		displayName := metricDisplayNames[c.name]
		if displayName == "" {
			displayName = c.name
		}
		strengths[i] = models.StrengthConcern{
			Metric:     c.name,
			Value:      floatPtr(c.val),
			Percentile: floatPtr(c.pct),
			Message:    fmt.Sprintf("%s ranks in top %.0f%% of %s sector", displayName, 100-c.pct, sector),
		}
	}

	return strengths
}

func generateConcerns(metricsMap map[string]*float64, percentileMap map[string]*float64, sector string, limit int) []models.StrengthConcern {
	type metricPct struct {
		name string
		pct  float64
		val  float64
	}

	var candidates []metricPct
	for name, pct := range percentileMap {
		if pct == nil || *pct > 25 {
			continue
		}
		val := metricsMap[name]
		v := 0.0
		if val != nil {
			v = *val
		}
		candidates = append(candidates, metricPct{name: name, pct: *pct, val: v})
	}

	// Sort by percentile ascending (worst first)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].pct < candidates[j].pct
	})

	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	concerns := make([]models.StrengthConcern, len(candidates))
	for i, c := range candidates {
		displayName := metricDisplayNames[c.name]
		if displayName == "" {
			displayName = c.name
		}
		concerns[i] = models.StrengthConcern{
			Metric:     c.name,
			Value:      floatPtr(c.val),
			Percentile: floatPtr(c.pct),
			Message:    fmt.Sprintf("%s is below %.0f%% of %s sector peers", displayName, 100-c.pct, sector),
		}
	}

	return concerns
}

// ============================================================================
// Red Flag Detection
// ============================================================================

func detectRedFlags(metrics *models.StockMetricsRow, fScore *int, zScore *float64, percentileMap map[string]*float64) []models.RedFlag {
	var flags []models.RedFlag

	// Altman Z-Score distress
	if zScore != nil && *zScore < 1.81 {
		flags = append(flags, models.RedFlag{
			ID:             "altman_distress",
			Severity:       "high",
			Title:          "Altman Z-Score indicates financial distress",
			Description:    fmt.Sprintf("Z-Score of %.2f is below the 1.81 distress threshold", *zScore),
			RelatedMetrics: []string{"altman_z_score"},
		})
	}

	// Weak Piotroski F-Score
	if fScore != nil && *fScore <= 3 {
		flags = append(flags, models.RedFlag{
			ID:             "weak_piotroski",
			Severity:       "medium",
			Title:          "Weak Piotroski F-Score",
			Description:    fmt.Sprintf("F-Score of %d/9 suggests deteriorating fundamentals", *fScore),
			RelatedMetrics: []string{"piotroski_f_score"},
		})
	}

	if metrics != nil {
		// Unsustainable dividend
		if metrics.PayoutRatio != nil && *metrics.PayoutRatio > 100 {
			flags = append(flags, models.RedFlag{
				ID:             "unsustainable_dividend",
				Severity:       "high",
				Title:          "Unsustainable dividend payout",
				Description:    fmt.Sprintf("Payout ratio of %.1f%% exceeds 100%% — dividend may not be sustainable", *metrics.PayoutRatio),
				RelatedMetrics: []string{"payout_ratio", "dividend_yield"},
			})
		}

		// High leverage (D/E in bottom percentile AND low interest coverage)
		if metrics.DebtToEquity != nil && metrics.InterestCoverage != nil {
			dePct := percentileMap["debt_to_equity"]
			if dePct != nil && *dePct <= 10 && *metrics.InterestCoverage < 2 {
				flags = append(flags, models.RedFlag{
					ID:             "high_leverage",
					Severity:       "high",
					Title:          "High leverage with low interest coverage",
					Description:    fmt.Sprintf("Debt/Equity of %.2f with interest coverage of %.1fx", *metrics.DebtToEquity, *metrics.InterestCoverage),
					RelatedMetrics: []string{"debt_to_equity", "interest_coverage"},
				})
			}
		}
	}

	if flags == nil {
		flags = []models.RedFlag{}
	}

	return flags
}

// ============================================================================
// Trend Computation
// ============================================================================

func computeTrend(dataPoints []models.MetricDataPoint, timeframe string) *models.MetricTrend {
	if len(dataPoints) < 2 {
		return nil
	}

	// Direction: compare latest to previous
	latest := dataPoints[0].Value
	previous := dataPoints[1].Value
	direction := "flat"
	if latest != nil && previous != nil {
		if *latest > *previous {
			direction = "up"
		} else if *latest < *previous {
			direction = "down"
		}
	}

	// Consecutive growth periods (counting from most recent)
	consecutive := 0
	for i := 0; i < len(dataPoints)-1; i++ {
		cur := dataPoints[i].Value
		next := dataPoints[i+1].Value
		if cur != nil && next != nil && *cur > *next {
			consecutive++
		} else {
			break
		}
	}

	// Normalized slope: (newest - oldest) / periods / |oldest|
	var slope *float64
	first := dataPoints[len(dataPoints)-1].Value
	last := dataPoints[0].Value
	if first != nil && last != nil && math.Abs(*first) > 1e-10 {
		s := (*last - *first) / float64(len(dataPoints)-1) / math.Abs(*first)
		slope = &s
	}

	return &models.MetricTrend{
		Direction:                 direction,
		Slope:                     slope,
		ConsecutiveGrowthQuarters: consecutive,
	}
}

// ============================================================================
// Utility
// ============================================================================

func floatPtr(f float64) *float64 {
	return &f
}
