# Test Coverage Improvement Plan

## Current State

| Service | Language | Coverage | CI Threshold |
|---|---|---|---|
| Notification Service | Go | 81.7% | 80% |
| Frontend (Next.js) | TypeScript | 73.5% | 65% |
| IC Score Service | Python | 66.0% | 50% |
| Data Ingestion Service | Go | 35.5% | N/A |
| Task Service | Go | 35.5% | N/A |
| Backend API | Go | 21.0% | 20% |

## Root Cause: Go Services Can't Mock Network Dependencies

All three low-coverage Go services (backend, data-ingestion, task-service) share the same architectural problem:

1. **Global state instead of dependency injection** — `database.DB` is a package-level `*sqlx.DB` variable. `storage.s3Client` is a package-level `*s3.Client`. Handlers call these globals directly, making it impossible to substitute mocks in tests.
2. **No interfaces** — Functions like `database.InsertIngestionLog()` are package-level functions, not methods on an interface. You can't swap in a mock implementation.
3. **External services as hard requirements** — Handlers directly call `storage.Upload()`, `database.GetIngestionLogs()`, `emailService.Send()`, etc. without any abstraction boundary.

The result: even a simple handler test requires a real PostgreSQL database, real AWS S3 credentials, and real SMTP access — none of which exist in the test environment.

## The Plan

### Phase 1: Go Interface Extraction (Foundation — Enables Everything Else)

This is the critical unlock. Without interfaces, nothing else works.

#### 1A. Backend API — Introduce `Repository` and `ExternalService` interfaces

**File: `backend/database/interfaces.go`** (new)
```go
type UserRepository interface {
    CreateUser(ctx context.Context, user *models.User) error
    GetUserByID(ctx context.Context, id string) (*models.User, error)
    GetUserByEmail(ctx context.Context, email string) (*models.User, error)
    UpdateUser(ctx context.Context, user *models.User) error
    // ... remaining user operations
}

type WatchlistRepository interface {
    CreateWatchList(ctx context.Context, wl *models.WatchList) error
    GetWatchListsByUserID(ctx context.Context, userID string) ([]models.WatchList, error)
    // ...
}

type AlertRepository interface { ... }
type StockRepository interface { ... }
type FinancialRepository interface { ... }
// one interface per domain aggregate
```

**File: `backend/services/interfaces.go`** (new)
```go
type EmailSender interface {
    SendVerificationEmail(to, token string) error
    SendPasswordResetEmail(to, token string) error
}

type PriceProvider interface {
    GetQuote(symbol string) (*Quote, error)
    GetHistoricalData(symbol string, from, to time.Time) ([]Bar, error)
}
```

**File: `backend/handlers/handler.go`** (new — handler struct with injected deps)
```go
type Handler struct {
    Users      database.UserRepository
    Watchlists database.WatchlistRepository
    Alerts     database.AlertRepository
    Email      services.EmailSender
    Prices     services.PriceProvider
    // ...
}
```

Then refactor each handler from:
```go
func Login(c *gin.Context) {
    user, err := database.GetUserByEmail(email)
```
to:
```go
func (h *Handler) Login(c *gin.Context) {
    user, err := h.Users.GetUserByEmail(c.Request.Context(), email)
```

The existing concrete implementations (the current global functions) get wrapped to satisfy the interface. Production code stays identical in behavior — we're just making the seams explicit.

#### 1B. Data Ingestion Service — Same pattern

**Interfaces needed:**
- `StorageUploader` — wraps `storage.Upload()`, `storage.GenerateKey()`, `storage.GetBucket()`
- `IngestionLogStore` — wraps `database.InsertIngestionLog()`, `Get*` functions
- `SchemaValidator` — wraps `gojsonschema` calls (currently hardcoded `file://` paths)

**Refactor:** Create `Handler` struct in `handlers/handler.go` that accepts these interfaces.

#### 1C. Task Service — Same pattern

**Interfaces needed:**
- `TaskStore` — wraps all `database.DB.QueryRow/Exec` calls for tasks
- `TaskTypeStore` — wraps task type CRUD

---

### Phase 2: Go Test Infrastructure

With interfaces in place, create mock implementations.

#### 2A. Use `go-sqlmock` for repository tests

Test the concrete repository implementations (the ones that actually run SQL) against `go-sqlmock`. This covers the `database/` packages at 0-4.3% coverage.

```go
func TestInsertIngestionLog(t *testing.T) {
    db, mock, _ := sqlmock.New()
    repo := NewPostgresIngestionLogStore(sqlx.NewDb(db, "postgres"))

    mock.ExpectQuery("INSERT INTO ingestion_log").
        WithArgs(...).
        WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

    id, err := repo.InsertIngestionLog(...)
    assert.NoError(t, err)
    assert.Equal(t, int64(1), id)
}
```

**Target packages:**
- `backend/database/` — 4.3% → 60%+
- `data-ingestion-service/database/` — 0% → 60%+
- `task-service/database/` — 0% → 60%+

#### 2B. Use mock interfaces for handler tests

Test handlers with in-memory mock implementations of the interfaces. No real DB, no real S3.

```go
type mockStorage struct {
    uploaded map[string][]byte
}
func (m *mockStorage) Upload(key string, data []byte, ct string) error {
    m.uploaded[key] = data
    return nil
}

func TestPostIngest_Success(t *testing.T) {
    h := &Handler{
        Storage: &mockStorage{uploaded: map[string][]byte{}},
        DB:      &mockIngestionLogStore{},
    }
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    // ... set up request
    h.PostIngest(c)
    assert.Equal(t, 201, w.Code)
}
```

**Target packages:**
- `backend/handlers/` — 20.9% → 60%+
- `data-ingestion-service/handlers/` — 38.9% → 70%+
- `data-ingestion-service/handlers/ycharts/` — 0% → 60%+
- `task-service/handlers/` — 34.3% → 70%+

#### 2C. Use `httptest` for external API mocking in services

For `backend/services/` (Polygon, FMP, CoinGecko):
```go
func TestGetQuote(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        json.NewEncoder(w).Encode(polygonResponse{...})
    }))
    defer server.Close()

    client := NewPolygonClient(WithBaseURL(server.URL))
    quote, err := client.GetQuote("AAPL")
    // ...
}
```

**Target:** `backend/services/` — 40.5% → 65%+

---

### Phase 3: Quick Wins (No Refactoring Needed)

These can be done in parallel with Phase 1 since they don't require interface extraction.

#### 3A. Backend — Rate limiter tests (`auth/rate_limit.go`)
- Pure in-memory logic, no dependencies
- Test `Allow()`, `Cleanup()`, concurrent access
- Effort: ~1 hour
- Impact: auth package 61.9% → 75%+

#### 3B. Backend — `mapSICToSector()`, `nullIfEmpty()` in import-tickers
- Pure functions, trivial to test
- Impact: cmd/import-tickers 10.4% → 30%+

#### 3C. Task Service — `joinStrings()`, `getUserID()`, `scanTask()`
- Pure or interface-accepting helpers
- Impact: handlers 34.3% → 40%+

#### 3D. Data Ingestion — `LoadConfigFromEnv()`
- Set env vars, call function, assert struct
- Impact: database 0% → 15%+

#### 3E. IC Score Service — External API client tests
- `polygon_client.py` — 10 functions, 0% coverage, mock `requests.Session`
- `sec_client.py` — 12 functions, 0% coverage, mock EDGAR API responses
- Effort: ~8 hours
- Impact: overall 66% → 72%+

#### 3F. IC Score Service — `data_validator.py` and `dependency_checker.py`
- Pure validation logic, minimal mocking needed
- Effort: ~4 hours
- Impact: +2-3%

#### 3G. Frontend — `ic-score.ts` API module
- 0% coverage, 8 functions, mock fetch
- Follow existing pattern in `lib/api/__tests__/`
- Effort: ~2 hours
- Impact: +1-2%

#### 3H. Frontend — Untested hooks
- `useModal`, `useSlashFocus`, `useWatchlistAlerts`
- Effort: ~3 hours
- Impact: +1-2%

---

### Phase 4: Integration Tests (After Phase 1-2)

Once interfaces exist and unit tests pass:

#### 4A. Backend — Full request cycle tests
Test router → middleware → handler → mock repo, using `httptest.Server`:
- Auth flow: signup → verify email → login → refresh token
- Watchlist CRUD with auth
- Alert creation with notification

#### 4B. IC Score Service — Pipeline orchestration tests
- Mock database with `AsyncSession` patches
- Test `ic_score_calculator.run()` end-to-end with mock data
- Test score stabilization across multiple runs

#### 4C. Frontend — Component tests for highest-impact pages
Priority (by size/complexity):
1. `ICScoreMethodology.tsx` (662 lines)
2. `BacktestDashboard.tsx` (318 lines)
3. `ICScoreCardV2.tsx` (358 lines)
4. `FinancialsTab.tsx`

---

## Projected Coverage After Each Phase

| Service | Current | After Phase 1-2 | After Phase 3 | After Phase 4 |
|---|---|---|---|---|
| Backend API | 21.0% | 50%+ | 55%+ | 65%+ |
| Data Ingestion | 35.5% | 55%+ | 60%+ | 70%+ |
| Task Service | 35.5% | 55%+ | 60%+ | 70%+ |
| IC Score Service | 66.0% | 66% | 75%+ | 80%+ |
| Frontend | 73.5% | 73.5% | 77%+ | 82%+ |
| Notification | 81.7% | 81.7% | 81.7% | 81.7% |

## Priority Order

1. **Phase 1A+2B** (Backend handler interfaces + mock tests) — Biggest coverage delta, most business logic lives here
2. **Phase 3E** (IC Score API clients) — Easy wins, high risk area (external APIs)
3. **Phase 1B+1C** (Data ingestion + Task service interfaces) — Same pattern as 1A, smaller codebase
4. **Phase 3A-3D** (Pure function quick wins) — Can be done by anyone, no design decisions
5. **Phase 3G-3H** (Frontend quick wins) — Already at 73.5%, diminishing returns
6. **Phase 4** (Integration tests) — Builds on everything above
