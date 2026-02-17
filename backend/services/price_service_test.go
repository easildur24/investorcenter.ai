package services

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// getFloat64 — pure helper function
// ---------------------------------------------------------------------------

func TestGetFloat64_Valid(t *testing.T) {
	n := sql.NullFloat64{Float64: 42.5, Valid: true}
	assert.Equal(t, 42.5, getFloat64(n))
}

func TestGetFloat64_Invalid(t *testing.T) {
	n := sql.NullFloat64{Float64: 99.9, Valid: false}
	assert.Equal(t, float64(0), getFloat64(n))
}

func TestGetFloat64_Zero(t *testing.T) {
	n := sql.NullFloat64{Float64: 0, Valid: true}
	assert.Equal(t, float64(0), getFloat64(n))
}

func TestGetFloat64_Negative(t *testing.T) {
	n := sql.NullFloat64{Float64: -100.5, Valid: true}
	assert.Equal(t, -100.5, getFloat64(n))
}

// ---------------------------------------------------------------------------
// getInt64 — pure helper function
// ---------------------------------------------------------------------------

func TestGetInt64_Valid(t *testing.T) {
	n := sql.NullInt64{Int64: 1000000, Valid: true}
	assert.Equal(t, int64(1000000), getInt64(n))
}

func TestGetInt64_Invalid(t *testing.T) {
	n := sql.NullInt64{Int64: 999, Valid: false}
	assert.Equal(t, int64(0), getInt64(n))
}

func TestGetInt64_Zero(t *testing.T) {
	n := sql.NullInt64{Int64: 0, Valid: true}
	assert.Equal(t, int64(0), getInt64(n))
}

func TestGetInt64_Negative(t *testing.T) {
	n := sql.NullInt64{Int64: -500, Valid: true}
	assert.Equal(t, int64(-500), getInt64(n))
}
