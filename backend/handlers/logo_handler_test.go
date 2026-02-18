package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"

	"investorcenter-api/database"
)

// skipIfNoTestDB skips logo tests unless INTEGRATION_TEST_DB is set.
func skipIfNoTestDB(t *testing.T) {
	t.Helper()
	if os.Getenv("INTEGRATION_TEST_DB") != "true" {
		t.Skip("Skipping integration test: INTEGRATION_TEST_DB not set")
	}
}

// ensureTestDB connects to the test database and swaps database.DB.
// Must be called after skipIfNoTestDB.
func ensureTestDB(t *testing.T) {
	t.Helper()
	if database.DB != nil {
		return // already connected
	}

	host := envOrDefault("DB_HOST", "localhost")
	port := envOrDefault("DB_PORT", "5432")
	user := envOrDefault("DB_USER", "testuser")
	pass := envOrDefault("DB_PASSWORD", "testpass")
	name := envOrDefault("DB_NAME", "investorcenter_test")
	sslmode := envOrDefault("DB_SSLMODE", "disable")

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, pass, name, sslmode,
	)

	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		t.Fatalf("failed to connect to test DB: %v", err)
	}

	origDB := database.DB
	database.DB = db
	t.Cleanup(func() {
		db.Close()
		database.DB = origDB
	})
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func setupLogoRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/logos/:symbol", ProxyLogo)
	return r
}

func TestProxyLogo_StockNotFound(t *testing.T) {
	skipIfNoTestDB(t)
	ensureTestDB(t)
	router := setupLogoRouter()

	// With DB connected but no stock data, should 404
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/logos/ZZZZZZZZ", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestProxyLogo_NoLogoURL(t *testing.T) {
	skipIfNoTestDB(t)
	ensureTestDB(t)
	router := setupLogoRouter()

	// Insert a ticker with no logo URL
	database.DB.MustExec(`INSERT INTO tickers (symbol, name, asset_type, logo_url) VALUES ('NOLOG', 'No Logo Corp', 'stock', '') ON CONFLICT (symbol) DO NOTHING`)
	t.Cleanup(func() {
		database.DB.Exec("DELETE FROM tickers WHERE symbol = 'NOLOG'")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/logos/NOLOG", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestProxyLogo_NoAPIKey(t *testing.T) {
	skipIfNoTestDB(t)
	ensureTestDB(t)
	router := setupLogoRouter()

	// Insert a ticker with a logo URL
	database.DB.MustExec(`INSERT INTO tickers (symbol, name, asset_type, logo_url) VALUES ('KEYTEST', 'Key Test Corp', 'stock', 'https://example.com/logo.png') ON CONFLICT (symbol) DO NOTHING`)
	t.Cleanup(func() {
		database.DB.Exec("DELETE FROM tickers WHERE symbol = 'KEYTEST'")
	})

	// Clear POLYGON_API_KEY
	origKey := os.Getenv("POLYGON_API_KEY")
	os.Unsetenv("POLYGON_API_KEY")
	t.Cleanup(func() {
		if origKey != "" {
			os.Setenv("POLYGON_API_KEY", origKey)
		}
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/logos/KEYTEST", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestProxyLogo_Success(t *testing.T) {
	skipIfNoTestDB(t)
	ensureTestDB(t)

	// Create a fake Polygon logo server
	fakePolygon := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify API key is appended
		assert.NotEmpty(t, r.URL.Query().Get("apiKey"))
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fake-png-data"))
	}))
	defer fakePolygon.Close()

	// Insert ticker pointing to our fake server
	database.DB.MustExec(`INSERT INTO tickers (symbol, name, asset_type, logo_url) VALUES ('LOGOTEST', 'Logo Test Corp', 'stock', $1) ON CONFLICT (symbol) DO UPDATE SET logo_url = $1`, fakePolygon.URL+"/logo.png")
	t.Cleanup(func() {
		database.DB.Exec("DELETE FROM tickers WHERE symbol = 'LOGOTEST'")
	})

	// Set API key
	origKey := os.Getenv("POLYGON_API_KEY")
	os.Setenv("POLYGON_API_KEY", "test_key_123")
	t.Cleanup(func() {
		if origKey != "" {
			os.Setenv("POLYGON_API_KEY", origKey)
		} else {
			os.Unsetenv("POLYGON_API_KEY")
		}
	})

	router := setupLogoRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/logos/LOGOTEST", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "public, max-age=86400", w.Header().Get("Cache-Control"))
	assert.Equal(t, "image/png", w.Header().Get("Content-Type"))
	assert.Equal(t, "fake-png-data", w.Body.String())
}

func TestProxyLogo_UpstreamError(t *testing.T) {
	skipIfNoTestDB(t)
	ensureTestDB(t)

	// Create a fake Polygon server that returns 500
	fakePolygon := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer fakePolygon.Close()

	// Insert ticker pointing to our fake server
	database.DB.MustExec(`INSERT INTO tickers (symbol, name, asset_type, logo_url) VALUES ('ERRLOGO', 'Error Logo Corp', 'stock', $1) ON CONFLICT (symbol) DO UPDATE SET logo_url = $1`, fakePolygon.URL+"/logo.png")
	t.Cleanup(func() {
		database.DB.Exec("DELETE FROM tickers WHERE symbol = 'ERRLOGO'")
	})

	origKey := os.Getenv("POLYGON_API_KEY")
	os.Setenv("POLYGON_API_KEY", "test_key_123")
	t.Cleanup(func() {
		if origKey != "" {
			os.Setenv("POLYGON_API_KEY", origKey)
		} else {
			os.Unsetenv("POLYGON_API_KEY")
		}
	})

	router := setupLogoRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/logos/ERRLOGO", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
