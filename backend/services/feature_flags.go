package services

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"
)

// FeatureFlagService manages feature flags for gradual rollout
type FeatureFlagService struct {
	flags     map[string]*FeatureFlag
	mu        sync.RWMutex
	configDir string
}

// FeatureFlag represents a single feature flag
type FeatureFlag struct {
	Name        string    `json:"name"`
	Enabled     bool      `json:"enabled"`
	Percentage  float64   `json:"percentage"` // 0-100, percentage of users
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// IC Score v2.1 Feature Flag Names
const (
	FlagSectorRelativeScoring     = "ic_score_sector_relative_scoring"
	FlagLifecycleClassification   = "ic_score_lifecycle_classification"
	FlagEarningsRevisionsFactor   = "ic_score_earnings_revisions_factor"
	FlagHistoricalValuationFactor = "ic_score_historical_valuation_factor"
	FlagDividendQualityFactor     = "ic_score_dividend_quality_factor"
	FlagScoreStability            = "ic_score_score_stability"
	FlagPeerComparison            = "ic_score_peer_comparison"
	FlagCatalysts                 = "ic_score_catalysts"
	FlagScoreChangeExplanations   = "ic_score_score_change_explanations"
	FlagGranularConfidence        = "ic_score_granular_confidence"
	FlagBacktestDashboard         = "ic_score_backtest_dashboard"
)

// NewFeatureFlagService creates a new feature flag service
func NewFeatureFlagService() *FeatureFlagService {
	service := &FeatureFlagService{
		flags:     make(map[string]*FeatureFlag),
		configDir: os.Getenv("FEATURE_FLAGS_DIR"),
	}

	// Initialize default flags
	service.initializeDefaultFlags()

	// Load from config file if exists
	if err := service.loadFromFile(); err != nil {
		log.Printf("Failed to load feature flags from file: %v", err)
	}

	return service
}

// initializeDefaultFlags sets up IC Score v2.1 feature flags
func (s *FeatureFlagService) initializeDefaultFlags() {
	now := time.Now()

	defaults := []*FeatureFlag{
		{
			Name:        FlagSectorRelativeScoring,
			Enabled:     false,
			Percentage:  0,
			Description: "Enable sector-relative scoring in IC Score calculation",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			Name:        FlagLifecycleClassification,
			Enabled:     false,
			Percentage:  0,
			Description: "Enable lifecycle stage classification for weight adjustments",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			Name:        FlagEarningsRevisionsFactor,
			Enabled:     false,
			Percentage:  0,
			Description: "Include earnings revisions in IC Score calculation",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			Name:        FlagHistoricalValuationFactor,
			Enabled:     false,
			Percentage:  0,
			Description: "Include historical valuation factor in IC Score",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			Name:        FlagDividendQualityFactor,
			Enabled:     false,
			Percentage:  0,
			Description: "Include optional dividend quality factor",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			Name:        FlagScoreStability,
			Enabled:     false,
			Percentage:  0,
			Description: "Apply score smoothing to reduce daily volatility",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			Name:        FlagPeerComparison,
			Enabled:     false,
			Percentage:  0,
			Description: "Show peer comparison in IC Score display",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			Name:        FlagCatalysts,
			Enabled:     false,
			Percentage:  0,
			Description: "Show upcoming catalysts in IC Score display",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			Name:        FlagScoreChangeExplanations,
			Enabled:     false,
			Percentage:  0,
			Description: "Show explanations for score changes",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			Name:        FlagGranularConfidence,
			Enabled:     false,
			Percentage:  0,
			Description: "Show per-factor data availability in confidence",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			Name:        FlagBacktestDashboard,
			Enabled:     false,
			Percentage:  0,
			Description: "Enable backtest results dashboard",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	for _, flag := range defaults {
		s.flags[flag.Name] = flag
	}
}

// IsEnabled checks if a feature is enabled for a given user
func (s *FeatureFlagService) IsEnabled(flagName string, userID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	flag, exists := s.flags[flagName]
	if !exists {
		return false
	}

	if !flag.Enabled {
		return false
	}

	// If percentage is 100, it's enabled for everyone
	if flag.Percentage >= 100 {
		return true
	}

	// If percentage is 0, it's disabled for everyone
	if flag.Percentage <= 0 {
		return false
	}

	// Use consistent hashing based on userID to determine if user is in rollout
	return s.isUserInPercentage(userID, flag.Percentage)
}

// isUserInPercentage determines if a user falls within the rollout percentage
func (s *FeatureFlagService) isUserInPercentage(userID string, percentage float64) bool {
	// Simple hash-based bucketing
	hash := 0
	for _, c := range userID {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}

	bucket := hash % 100
	return float64(bucket) < percentage
}

// GetFlag retrieves a feature flag by name
func (s *FeatureFlagService) GetFlag(name string) *FeatureFlag {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if flag, exists := s.flags[name]; exists {
		// Return a copy to prevent mutation
		copy := *flag
		return &copy
	}
	return nil
}

// GetAllFlags returns all feature flags
func (s *FeatureFlagService) GetAllFlags() []*FeatureFlag {
	s.mu.RLock()
	defer s.mu.RUnlock()

	flags := make([]*FeatureFlag, 0, len(s.flags))
	for _, flag := range s.flags {
		copy := *flag
		flags = append(flags, &copy)
	}
	return flags
}

// SetFlag updates or creates a feature flag
func (s *FeatureFlagService) SetFlag(flag *FeatureFlag) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	flag.UpdatedAt = time.Now()
	if existing, exists := s.flags[flag.Name]; exists {
		flag.CreatedAt = existing.CreatedAt
	} else {
		flag.CreatedAt = flag.UpdatedAt
	}

	s.flags[flag.Name] = flag

	// Persist to file
	return s.saveToFile()
}

// EnableFlag enables a flag for a percentage of users
func (s *FeatureFlagService) EnableFlag(name string, percentage float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	flag, exists := s.flags[name]
	if !exists {
		return nil // Silently ignore unknown flags
	}

	flag.Enabled = true
	flag.Percentage = percentage
	flag.UpdatedAt = time.Now()

	return s.saveToFile()
}

// DisableFlag disables a flag completely
func (s *FeatureFlagService) DisableFlag(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	flag, exists := s.flags[name]
	if !exists {
		return nil
	}

	flag.Enabled = false
	flag.Percentage = 0
	flag.UpdatedAt = time.Now()

	return s.saveToFile()
}

// GetEnabledFeaturesForUser returns list of enabled features for a user
func (s *FeatureFlagService) GetEnabledFeaturesForUser(userID string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	enabled := make([]string, 0)
	for name, flag := range s.flags {
		if flag.Enabled && (flag.Percentage >= 100 || s.isUserInPercentage(userID, flag.Percentage)) {
			enabled = append(enabled, name)
		}
	}
	return enabled
}

// loadFromFile loads flags from config file
func (s *FeatureFlagService) loadFromFile() error {
	if s.configDir == "" {
		return nil
	}

	filepath := s.configDir + "/feature_flags.json"
	data, err := os.ReadFile(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist, use defaults
		}
		return err
	}

	var flags []*FeatureFlag
	if err := json.Unmarshal(data, &flags); err != nil {
		return err
	}

	for _, flag := range flags {
		s.flags[flag.Name] = flag
	}

	return nil
}

// saveToFile persists flags to config file
func (s *FeatureFlagService) saveToFile() error {
	if s.configDir == "" {
		return nil
	}

	flags := make([]*FeatureFlag, 0, len(s.flags))
	for _, flag := range s.flags {
		flags = append(flags, flag)
	}

	data, err := json.MarshalIndent(flags, "", "  ")
	if err != nil {
		return err
	}

	filepath := s.configDir + "/feature_flags.json"
	return os.WriteFile(filepath, data, 0644)
}

// RolloutPhase represents a rollout phase configuration
type RolloutPhase struct {
	Name       string   `json:"name"`
	Percentage float64  `json:"percentage"`
	Features   []string `json:"features"`
}

// GetRolloutPhases returns the IC Score v2.1 rollout phases
func (s *FeatureFlagService) GetRolloutPhases() []RolloutPhase {
	return []RolloutPhase{
		{
			Name:       "Alpha",
			Percentage: 1,
			Features: []string{
				FlagSectorRelativeScoring,
				FlagLifecycleClassification,
				FlagEarningsRevisionsFactor,
				FlagHistoricalValuationFactor,
				FlagScoreStability,
				FlagPeerComparison,
				FlagCatalysts,
				FlagScoreChangeExplanations,
				FlagGranularConfidence,
				FlagBacktestDashboard,
			},
		},
		{
			Name:       "Beta",
			Percentage: 5,
			Features: []string{
				FlagSectorRelativeScoring,
				FlagLifecycleClassification,
				FlagEarningsRevisionsFactor,
				FlagHistoricalValuationFactor,
				FlagScoreStability,
				FlagPeerComparison,
				FlagCatalysts,
				FlagScoreChangeExplanations,
				FlagGranularConfidence,
				FlagBacktestDashboard,
			},
		},
		{
			Name:       "GA Phase 1",
			Percentage: 25,
			Features: []string{
				FlagSectorRelativeScoring,
				FlagLifecycleClassification,
				FlagEarningsRevisionsFactor,
				FlagHistoricalValuationFactor,
				FlagScoreStability,
				FlagPeerComparison,
				FlagCatalysts,
				FlagScoreChangeExplanations,
				FlagGranularConfidence,
				FlagBacktestDashboard,
			},
		},
		{
			Name:       "GA Phase 2",
			Percentage: 50,
			Features: []string{
				FlagSectorRelativeScoring,
				FlagLifecycleClassification,
				FlagEarningsRevisionsFactor,
				FlagHistoricalValuationFactor,
				FlagScoreStability,
				FlagPeerComparison,
				FlagCatalysts,
				FlagScoreChangeExplanations,
				FlagGranularConfidence,
				FlagBacktestDashboard,
			},
		},
		{
			Name:       "GA Phase 3",
			Percentage: 100,
			Features: []string{
				FlagSectorRelativeScoring,
				FlagLifecycleClassification,
				FlagEarningsRevisionsFactor,
				FlagHistoricalValuationFactor,
				FlagScoreStability,
				FlagPeerComparison,
				FlagCatalysts,
				FlagScoreChangeExplanations,
				FlagGranularConfidence,
				FlagBacktestDashboard,
			},
		},
	}
}

// ApplyRolloutPhase applies a rollout phase configuration
func (s *FeatureFlagService) ApplyRolloutPhase(phaseName string) error {
	phases := s.GetRolloutPhases()

	for _, phase := range phases {
		if phase.Name == phaseName {
			for _, feature := range phase.Features {
				if err := s.EnableFlag(feature, phase.Percentage); err != nil {
					return err
				}
			}
			return nil
		}
	}

	return nil
}

// RollbackAll disables all IC Score v2.1 features (emergency rollback)
func (s *FeatureFlagService) RollbackAll() error {
	allFlags := []string{
		FlagSectorRelativeScoring,
		FlagLifecycleClassification,
		FlagEarningsRevisionsFactor,
		FlagHistoricalValuationFactor,
		FlagDividendQualityFactor,
		FlagScoreStability,
		FlagPeerComparison,
		FlagCatalysts,
		FlagScoreChangeExplanations,
		FlagGranularConfidence,
		FlagBacktestDashboard,
	}

	for _, flag := range allFlags {
		if err := s.DisableFlag(flag); err != nil {
			return err
		}
	}

	return nil
}
