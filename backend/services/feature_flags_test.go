package services

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestFlagService creates a FeatureFlagService with no config dir (no file I/O).
func newTestFlagService() *FeatureFlagService {
	os.Setenv("FEATURE_FLAGS_DIR", "")
	svc := &FeatureFlagService{
		flags:     make(map[string]*FeatureFlag),
		configDir: "",
	}
	svc.initializeDefaultFlags()
	return svc
}

// ---------------------------------------------------------------------------
// Default flag initialization
// ---------------------------------------------------------------------------

func TestDefaultFlagsInitialized(t *testing.T) {
	svc := newTestFlagService()
	flags := svc.GetAllFlags()

	assert.GreaterOrEqual(t, len(flags), 11, "should have at least 11 default flags")

	// Every default flag should be disabled with 0% rollout
	for _, f := range flags {
		assert.False(t, f.Enabled, "default flag %s should be disabled", f.Name)
		assert.Equal(t, float64(0), f.Percentage, "default flag %s should have 0%% rollout", f.Name)
	}
}

func TestGetFlag_Exists(t *testing.T) {
	svc := newTestFlagService()

	flag := svc.GetFlag(FlagBacktestDashboard)
	require.NotNil(t, flag)
	assert.Equal(t, FlagBacktestDashboard, flag.Name)
}

func TestGetFlag_NonExistent(t *testing.T) {
	svc := newTestFlagService()

	flag := svc.GetFlag("nonexistent_flag")
	assert.Nil(t, flag)
}

func TestGetFlag_ReturnsCopy(t *testing.T) {
	svc := newTestFlagService()

	flag := svc.GetFlag(FlagBacktestDashboard)
	require.NotNil(t, flag)

	// Mutating the copy should not affect the original
	flag.Enabled = true
	flag.Percentage = 100

	original := svc.GetFlag(FlagBacktestDashboard)
	require.NotNil(t, original)
	assert.False(t, original.Enabled, "mutating copy should not change original")
	assert.Equal(t, float64(0), original.Percentage)
}

// ---------------------------------------------------------------------------
// IsEnabled logic
// ---------------------------------------------------------------------------

func TestIsEnabled_UnknownFlag(t *testing.T) {
	svc := newTestFlagService()
	assert.False(t, svc.IsEnabled("totally_unknown", "user-1"))
}

func TestIsEnabled_DisabledFlag(t *testing.T) {
	svc := newTestFlagService()
	// All defaults are disabled
	assert.False(t, svc.IsEnabled(FlagBacktestDashboard, "user-1"))
}

func TestIsEnabled_EnabledAt100Percent(t *testing.T) {
	svc := newTestFlagService()

	svc.flags[FlagBacktestDashboard].Enabled = true
	svc.flags[FlagBacktestDashboard].Percentage = 100

	assert.True(t, svc.IsEnabled(FlagBacktestDashboard, "user-1"))
	assert.True(t, svc.IsEnabled(FlagBacktestDashboard, "user-2"))
	assert.True(t, svc.IsEnabled(FlagBacktestDashboard, "any-user"))
}

func TestIsEnabled_EnabledAt0Percent(t *testing.T) {
	svc := newTestFlagService()

	svc.flags[FlagBacktestDashboard].Enabled = true
	svc.flags[FlagBacktestDashboard].Percentage = 0

	assert.False(t, svc.IsEnabled(FlagBacktestDashboard, "user-1"))
}

func TestIsEnabled_PartialRollout(t *testing.T) {
	svc := newTestFlagService()

	svc.flags[FlagBacktestDashboard].Enabled = true
	svc.flags[FlagBacktestDashboard].Percentage = 50

	// With 50% rollout, the hash-based bucketing should be deterministic.
	// The same user ID should always get the same result.
	result1 := svc.IsEnabled(FlagBacktestDashboard, "consistent-user")
	result2 := svc.IsEnabled(FlagBacktestDashboard, "consistent-user")
	assert.Equal(t, result1, result2, "result should be deterministic for same user")
}

// ---------------------------------------------------------------------------
// isUserInPercentage — determinism and bucketing
// ---------------------------------------------------------------------------

func TestIsUserInPercentage_Deterministic(t *testing.T) {
	svc := newTestFlagService()

	for _, userID := range []string{"alice", "bob", "charlie", "user-42"} {
		r1 := svc.isUserInPercentage(userID, 50)
		r2 := svc.isUserInPercentage(userID, 50)
		assert.Equal(t, r1, r2, "should be deterministic for user %s", userID)
	}
}

func TestIsUserInPercentage_ZeroPercent(t *testing.T) {
	svc := newTestFlagService()
	assert.False(t, svc.isUserInPercentage("any-user", 0))
}

func TestIsUserInPercentage_HundredPercent(t *testing.T) {
	svc := newTestFlagService()
	assert.True(t, svc.isUserInPercentage("any-user", 100))
}

func TestIsUserInPercentage_Distribution(t *testing.T) {
	svc := newTestFlagService()

	// With many users and 50% rollout, we expect roughly half to be in
	enabledCount := 0
	total := 1000
	for i := 0; i < total; i++ {
		userID := "user-" + string(rune(i+'A'))
		if svc.isUserInPercentage(userID, 50) {
			enabledCount++
		}
	}

	// Allow wide margin: just check it's not all-or-nothing
	assert.Greater(t, enabledCount, 0, "should have some users enabled at 50%%")
	assert.Less(t, enabledCount, total, "should not have all users enabled at 50%%")
}

// ---------------------------------------------------------------------------
// EnableFlag / DisableFlag
// ---------------------------------------------------------------------------

func TestEnableFlag(t *testing.T) {
	svc := newTestFlagService()

	err := svc.EnableFlag(FlagBacktestDashboard, 50)
	require.NoError(t, err)

	flag := svc.GetFlag(FlagBacktestDashboard)
	require.NotNil(t, flag)
	assert.True(t, flag.Enabled)
	assert.Equal(t, float64(50), flag.Percentage)
}

func TestEnableFlag_UnknownFlag(t *testing.T) {
	svc := newTestFlagService()

	// Unknown flags are silently ignored
	err := svc.EnableFlag("unknown_flag", 50)
	assert.NoError(t, err)
}

func TestDisableFlag(t *testing.T) {
	svc := newTestFlagService()

	// Enable first
	svc.flags[FlagBacktestDashboard].Enabled = true
	svc.flags[FlagBacktestDashboard].Percentage = 100

	err := svc.DisableFlag(FlagBacktestDashboard)
	require.NoError(t, err)

	flag := svc.GetFlag(FlagBacktestDashboard)
	require.NotNil(t, flag)
	assert.False(t, flag.Enabled)
	assert.Equal(t, float64(0), flag.Percentage)
}

func TestDisableFlag_UnknownFlag(t *testing.T) {
	svc := newTestFlagService()
	err := svc.DisableFlag("unknown_flag")
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// SetFlag
// ---------------------------------------------------------------------------

func TestSetFlag_NewFlag(t *testing.T) {
	svc := newTestFlagService()

	flag := &FeatureFlag{
		Name:        "custom_flag",
		Enabled:     true,
		Percentage:  75,
		Description: "Custom test flag",
	}

	err := svc.SetFlag(flag)
	require.NoError(t, err)

	retrieved := svc.GetFlag("custom_flag")
	require.NotNil(t, retrieved)
	assert.True(t, retrieved.Enabled)
	assert.Equal(t, float64(75), retrieved.Percentage)
	assert.Equal(t, "Custom test flag", retrieved.Description)
}

func TestSetFlag_UpdateExisting(t *testing.T) {
	svc := newTestFlagService()

	original := svc.GetFlag(FlagBacktestDashboard)
	require.NotNil(t, original)
	originalCreatedAt := original.CreatedAt

	updated := &FeatureFlag{
		Name:        FlagBacktestDashboard,
		Enabled:     true,
		Percentage:  25,
		Description: "Updated description",
	}

	err := svc.SetFlag(updated)
	require.NoError(t, err)

	result := svc.GetFlag(FlagBacktestDashboard)
	require.NotNil(t, result)
	assert.True(t, result.Enabled)
	assert.Equal(t, float64(25), result.Percentage)
	// CreatedAt should be preserved from original
	assert.Equal(t, originalCreatedAt, result.CreatedAt)
}

// ---------------------------------------------------------------------------
// GetEnabledFeaturesForUser
// ---------------------------------------------------------------------------

func TestGetEnabledFeaturesForUser_NoneEnabled(t *testing.T) {
	svc := newTestFlagService()

	enabled := svc.GetEnabledFeaturesForUser("user-1")
	assert.Empty(t, enabled)
}

func TestGetEnabledFeaturesForUser_AllAt100Percent(t *testing.T) {
	svc := newTestFlagService()

	// Enable all flags at 100%
	for _, flag := range svc.flags {
		flag.Enabled = true
		flag.Percentage = 100
	}

	enabled := svc.GetEnabledFeaturesForUser("user-1")
	assert.Equal(t, len(svc.flags), len(enabled))
}

// ---------------------------------------------------------------------------
// GetRolloutPhases
// ---------------------------------------------------------------------------

func TestGetRolloutPhases(t *testing.T) {
	svc := newTestFlagService()

	phases := svc.GetRolloutPhases()
	require.Len(t, phases, 5)

	assert.Equal(t, "Alpha", phases[0].Name)
	assert.Equal(t, float64(1), phases[0].Percentage)

	assert.Equal(t, "Beta", phases[1].Name)
	assert.Equal(t, float64(5), phases[1].Percentage)

	assert.Equal(t, "GA Phase 3", phases[4].Name)
	assert.Equal(t, float64(100), phases[4].Percentage)

	// Each phase should have features
	for _, phase := range phases {
		assert.NotEmpty(t, phase.Features, "phase %s should have features", phase.Name)
	}
}

// ---------------------------------------------------------------------------
// ApplyRolloutPhase
// ---------------------------------------------------------------------------

func TestApplyRolloutPhase_Alpha(t *testing.T) {
	svc := newTestFlagService()

	err := svc.ApplyRolloutPhase("Alpha")
	require.NoError(t, err)

	// After Alpha phase, listed features should be enabled at 1%
	flag := svc.GetFlag(FlagSectorRelativeScoring)
	require.NotNil(t, flag)
	assert.True(t, flag.Enabled)
	assert.Equal(t, float64(1), flag.Percentage)
}

func TestApplyRolloutPhase_UnknownPhase(t *testing.T) {
	svc := newTestFlagService()

	// Unknown phase silently does nothing
	err := svc.ApplyRolloutPhase("NonexistentPhase")
	assert.NoError(t, err)

	// Flags should remain unchanged
	for _, flag := range svc.GetAllFlags() {
		assert.False(t, flag.Enabled)
	}
}

// ---------------------------------------------------------------------------
// RollbackAll
// ---------------------------------------------------------------------------

func TestRollbackAll(t *testing.T) {
	svc := newTestFlagService()

	// Enable several flags
	svc.flags[FlagSectorRelativeScoring].Enabled = true
	svc.flags[FlagSectorRelativeScoring].Percentage = 100
	svc.flags[FlagBacktestDashboard].Enabled = true
	svc.flags[FlagBacktestDashboard].Percentage = 50

	err := svc.RollbackAll()
	require.NoError(t, err)

	// All flags should be disabled
	for _, flag := range svc.GetAllFlags() {
		assert.False(t, flag.Enabled, "flag %s should be disabled after rollback", flag.Name)
		assert.Equal(t, float64(0), flag.Percentage, "flag %s should have 0%% after rollback", flag.Name)
	}
}

// ---------------------------------------------------------------------------
// File I/O edge cases (no config dir)
// ---------------------------------------------------------------------------

func TestLoadFromFile_NoConfigDir(t *testing.T) {
	svc := &FeatureFlagService{
		flags:     make(map[string]*FeatureFlag),
		configDir: "",
	}

	err := svc.loadFromFile()
	assert.NoError(t, err, "should not error when configDir is empty")
}

func TestSaveToFile_NoConfigDir(t *testing.T) {
	svc := &FeatureFlagService{
		flags:     make(map[string]*FeatureFlag),
		configDir: "",
	}

	err := svc.saveToFile()
	assert.NoError(t, err, "should not error when configDir is empty")
}

// ---------------------------------------------------------------------------
// Flag constant names
// ---------------------------------------------------------------------------

func TestFlagConstants(t *testing.T) {
	constants := []string{
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

	for _, c := range constants {
		assert.NotEmpty(t, c, "flag constant should not be empty")
		assert.Contains(t, c, "ic_score_", "flag constants should have ic_score_ prefix")
	}
}
