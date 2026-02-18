package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
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

func setupLogoRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/logos/:symbol", ProxyLogo)
	return r
}

func TestProxyLogo_StockNotFound(t *testing.T) {
	skipIfNoTestDB(t)
	router := setupLogoRouter()

	// With DB connected but no stock data, should 404
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/logos/ZZZZZZZZ", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestProxyLogo_NoLogoURL(t *testing.T) {
	skipIfNoTestDB(t)
	router := setupLogoRouter()

	// Insert a ticker with no logo URL
	if database.DB != nil {
		database.DB.MustExec(`INSERT INTO tickers (symbol, name, asset_type, logo_url) VALUES ('NOLOG', 'No Logo Corp', 'stock', '') ON CONFLICT (symbol) DO NOTHING`)
		t.Cleanup(func() {
			database.DB.Exec("DELETE FROM tickers WHERE symbol = 'NOLOG'")
		})
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/logos/NOLOG", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestProxyLogo_NoAPIKey(t *testing.T) {
	skipIfNoTestDB(t)
	router := setupLogoRouter()

	// Insert a ticker with a logo URL
	if database.DB != nil {
		database.DB.MustExec(`INSERT INTO tickers (symbol, name, asset_type, logo_url) VALUES ('KEYTEST', 'Key Test Corp', 'stock', 'https://example.com/logo.png') ON CONFLICT (symbol) DO NOTHING`)
		t.Cleanup(func() {
			database.DB.Exec("DELETE FROM tickers WHERE symbol = 'KEYTEST'")
		})
	}

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
	if database.DB != nil {
		database.DB.MustExec(`INSERT INTO tickers (symbol, name, asset_type, logo_url) VALUES ('LOGOTEST', 'Logo Test Corp', 'stock', $1) ON CONFLICT (symbol) DO UPDATE SET logo_url = $1`, fakePolygon.URL+"/logo.png")
		t.Cleanup(func() {
			database.DB.Exec("DELETE FROM tickers WHERE symbol = 'LOGOTEST'")
		})
	}

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

	// Create a fake Polygon server that returns 500
	fakePolygon := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer fakePolygon.Close()

	// Insert ticker pointing to our fake server
	if database.DB != nil {
		database.DB.MustExec(`INSERT INTO tickers (symbol, name, asset_type, logo_url) VALUES ('ERRLOGO', 'Error Logo Corp', 'stock', $1) ON CONFLICT (symbol) DO UPDATE SET logo_url = $1`, fakePolygon.URL+"/logo.png")
		t.Cleanup(func() {
			database.DB.Exec("DELETE FROM tickers WHERE symbol = 'ERRLOGO'")
		})
	}

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
