package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"

	"investorcenter/backend/database"
	"investorcenter/backend/models"
)

// SectorPercentileService provides sector percentile operations with caching
type SectorPercentileService struct {
	redisClient *redis.Client
	cacheTTL    time.Duration
}

// CachedSectorPercentile is the cached format for sector percentile data
type CachedSectorPercentile struct {
	Sector       string   `json:"sector"`
	MetricName   string   `json:"metric_name"`
	CalculatedAt string   `json:"calculated_at"`
	MinValue     *float64 `json:"min_value"`
	P10Value     *float64 `json:"p10_value"`
	P25Value     *float64 `json:"p25_value"`
	P50Value     *float64 `json:"p50_value"`
	P75Value     *float64 `json:"p75_value"`
	P90Value     *float64 `json:"p90_value"`
	MaxValue     *float64 `json:"max_value"`
	MeanValue    *float64 `json:"mean_value"`
	StdDev       *float64 `json:"std_dev"`
	SampleCount  *int     `json:"sample_count"`
	CachedAt     string   `json:"cached_at"`
}

var sectorPercentileService *SectorPercentileService

// GetSectorPercentileService returns the singleton service instance
func GetSectorPercentileService() *SectorPercentileService {
	if sectorPercentileService == nil {
		sectorPercentileService = newSectorPercentileService()
	}
	return sectorPercentileService
}

// newSectorPercentileService creates a new service instance
func newSectorPercentileService() *SectorPercentileService {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       1, // Use DB 1 for sector percentiles (DB 0 is for crypto)
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed for sector percentiles: %v", err)
	}

	return &SectorPercentileService{
		redisClient: client,
		cacheTTL:    1 * time.Hour, // Cache for 1 hour (data updates daily)
	}
}

// cacheKey generates the Redis key for a sector/metric combination
func (s *SectorPercentileService) cacheKey(sector, metric string) string {
	return fmt.Sprintf("sector_percentile:%s:%s", sector, metric)
}

// sectorCacheKey generates the Redis key for all metrics in a sector
func (s *SectorPercentileService) sectorCacheKey(sector string) string {
	return fmt.Sprintf("sector_percentiles:%s", sector)
}

// GetPercentile retrieves percentile data with caching
func (s *SectorPercentileService) GetPercentile(ctx context.Context, sector, metric string) (*CachedSectorPercentile, error) {
	key := s.cacheKey(sector, metric)

	// Try cache first
	cached, err := s.getFromCache(ctx, key)
	if err == nil && cached != nil {
		return cached, nil
	}

	// Fetch from database
	sp, err := database.GetSectorPercentile(sector, metric)
	if err != nil {
		return nil, err
	}
	if sp == nil {
		return nil, nil
	}

	// Convert to cached format
	cachedSP := s.toCachedFormat(sp)

	// Store in cache
	if err := s.setInCache(ctx, key, cachedSP); err != nil {
		log.Printf("Warning: Failed to cache sector percentile %s/%s: %v", sector, metric, err)
	}

	return cachedSP, nil
}

// GetSectorPercentiles retrieves all percentiles for a sector with caching
func (s *SectorPercentileService) GetSectorPercentiles(ctx context.Context, sector string) ([]CachedSectorPercentile, error) {
	key := s.sectorCacheKey(sector)

	// Try cache first
	cached, err := s.getListFromCache(ctx, key)
	if err == nil && len(cached) > 0 {
		return cached, nil
	}

	// Fetch from database
	percentiles, err := database.GetSectorPercentiles(sector)
	if err != nil {
		return nil, err
	}

	// Convert to cached format
	cachedList := make([]CachedSectorPercentile, len(percentiles))
	for i, sp := range percentiles {
		cachedList[i] = *s.toCachedFormat(&sp)
	}

	// Store in cache
	if err := s.setListInCache(ctx, key, cachedList); err != nil {
		log.Printf("Warning: Failed to cache sector percentiles for %s: %v", sector, err)
	}

	return cachedList, nil
}

// CalculatePercentileScore calculates the percentile score for a value
func (s *SectorPercentileService) CalculatePercentileScore(ctx context.Context, sector, metric string, value float64) (*float64, error) {
	sp, err := s.GetPercentile(ctx, sector, metric)
	if err != nil {
		return nil, err
	}
	if sp == nil {
		return nil, nil
	}

	// Calculate raw percentile using piecewise linear interpolation
	rawPct := s.interpolatePercentile(value, sp)

	// Invert for "lower is better" metrics
	if models.LowerIsBetterMetrics[metric] {
		rawPct = 100 - rawPct
	}

	return &rawPct, nil
}

// interpolatePercentile performs piecewise linear interpolation
func (s *SectorPercentileService) interpolatePercentile(value float64, sp *CachedSectorPercentile) float64 {
	minVal := ptrFloat64(sp.MinValue, 0)
	p10 := ptrFloat64(sp.P10Value, minVal)
	p25 := ptrFloat64(sp.P25Value, p10)
	p50 := ptrFloat64(sp.P50Value, p25)
	p75 := ptrFloat64(sp.P75Value, p50)
	p90 := ptrFloat64(sp.P90Value, p75)
	maxVal := ptrFloat64(sp.MaxValue, p90)

	switch {
	case value <= minVal:
		return 0
	case value >= maxVal:
		return 100
	case value <= p10:
		return interpolate(value, minVal, p10, 0, 10)
	case value <= p25:
		return interpolate(value, p10, p25, 10, 25)
	case value <= p50:
		return interpolate(value, p25, p50, 25, 50)
	case value <= p75:
		return interpolate(value, p50, p75, 50, 75)
	case value <= p90:
		return interpolate(value, p75, p90, 75, 90)
	default:
		return interpolate(value, p90, maxVal, 90, 100)
	}
}

// InvalidateCache clears all sector percentile cache
func (s *SectorPercentileService) InvalidateCache(ctx context.Context) error {
	pattern := "sector_percentile*"
	iter := s.redisClient.Scan(ctx, 0, pattern, 0).Iterator()

	for iter.Next(ctx) {
		if err := s.redisClient.Del(ctx, iter.Val()).Err(); err != nil {
			return fmt.Errorf("failed to delete key %s: %w", iter.Val(), err)
		}
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to scan keys: %w", err)
	}

	return nil
}

// InvalidateSectorCache clears cache for a specific sector
func (s *SectorPercentileService) InvalidateSectorCache(ctx context.Context, sector string) error {
	pattern := fmt.Sprintf("sector_percentile*:%s:*", sector)
	iter := s.redisClient.Scan(ctx, 0, pattern, 0).Iterator()

	for iter.Next(ctx) {
		if err := s.redisClient.Del(ctx, iter.Val()).Err(); err != nil {
			return fmt.Errorf("failed to delete key %s: %w", iter.Val(), err)
		}
	}

	// Also delete the sector list cache
	s.redisClient.Del(ctx, s.sectorCacheKey(sector))

	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to scan keys: %w", err)
	}

	return nil
}

// Private helper methods

func (s *SectorPercentileService) toCachedFormat(sp *models.SectorPercentile) *CachedSectorPercentile {
	cached := &CachedSectorPercentile{
		Sector:       sp.Sector,
		MetricName:   sp.MetricName,
		CalculatedAt: sp.CalculatedAt.Format("2006-01-02"),
		SampleCount:  sp.SampleCount,
		CachedAt:     time.Now().Format(time.RFC3339),
	}

	if sp.MinValue != nil {
		v, _ := sp.MinValue.Float64()
		cached.MinValue = &v
	}
	if sp.P10Value != nil {
		v, _ := sp.P10Value.Float64()
		cached.P10Value = &v
	}
	if sp.P25Value != nil {
		v, _ := sp.P25Value.Float64()
		cached.P25Value = &v
	}
	if sp.P50Value != nil {
		v, _ := sp.P50Value.Float64()
		cached.P50Value = &v
	}
	if sp.P75Value != nil {
		v, _ := sp.P75Value.Float64()
		cached.P75Value = &v
	}
	if sp.P90Value != nil {
		v, _ := sp.P90Value.Float64()
		cached.P90Value = &v
	}
	if sp.MaxValue != nil {
		v, _ := sp.MaxValue.Float64()
		cached.MaxValue = &v
	}
	if sp.MeanValue != nil {
		v, _ := sp.MeanValue.Float64()
		cached.MeanValue = &v
	}
	if sp.StdDev != nil {
		v, _ := sp.StdDev.Float64()
		cached.StdDev = &v
	}

	return cached
}

func (s *SectorPercentileService) getFromCache(ctx context.Context, key string) (*CachedSectorPercentile, error) {
	data, err := s.redisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var cached CachedSectorPercentile
	if err := json.Unmarshal([]byte(data), &cached); err != nil {
		return nil, err
	}

	return &cached, nil
}

func (s *SectorPercentileService) setInCache(ctx context.Context, key string, data *CachedSectorPercentile) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return s.redisClient.Set(ctx, key, jsonData, s.cacheTTL).Err()
}

func (s *SectorPercentileService) getListFromCache(ctx context.Context, key string) ([]CachedSectorPercentile, error) {
	data, err := s.redisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var cached []CachedSectorPercentile
	if err := json.Unmarshal([]byte(data), &cached); err != nil {
		return nil, err
	}

	return cached, nil
}

func (s *SectorPercentileService) setListInCache(ctx context.Context, key string, data []CachedSectorPercentile) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return s.redisClient.Set(ctx, key, jsonData, s.cacheTTL).Err()
}

// Helper functions

func interpolate(value, lowVal, highVal, lowPct, highPct float64) float64 {
	if highVal == lowVal {
		return lowPct
	}
	ratio := (value - lowVal) / (highVal - lowVal)
	return lowPct + ratio*(highPct-lowPct)
}

func ptrFloat64(p *float64, defaultVal float64) float64 {
	if p == nil {
		return defaultVal
	}
	return *p
}
