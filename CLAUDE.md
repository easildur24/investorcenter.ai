# CLAUDE.md - InvestorCenter.ai

## Project Overview

InvestorCenter.ai is a full-stack financial analytics platform for stock/crypto data, IC Score (proprietary scoring), watchlists, alerts, and sentiment analysis. The repository is a polyrepo with multiple services sharing a single PostgreSQL database.

## Architecture

```
Frontend (Next.js 14)  -->  Go API (Gin)  -->  PostgreSQL + Redis
                       -->  IC Score Service (FastAPI)
                       -->  Task Service (Go)
                       -->  Data Ingestion Service (Python)
                       -->  Cronjob Monitor (Go)
```

**Services:**

| Service | Language | Port | Location |
|---------|----------|------|----------|
| Frontend | TypeScript/Next.js 14 | 3000 | `/` (root) |
| Backend API | Go/Gin | 8080 | `backend/` |
| IC Score Service | Python/FastAPI | 8001 | `ic-score-service/` |
| Task Service | Go | - | `task-service/` |
| Data Ingestion | Python | - | `data-ingestion-service/` |
| Cronjob Monitor | Go | - | `cronjob-monitor/` |

**Infrastructure:** AWS EKS, ECR, Terraform (`terraform/`), Kubernetes manifests (`k8s/`).

## Directory Structure

```
app/                    # Next.js App Router pages
components/             # React components (feature-grouped)
  ui/                   # Shared UI primitives
  ticker/               # Ticker detail page components
  ic-score/             # IC Score display/charts
  watchlist/            # Watchlist management
  financials/           # Financial statement views
  alerts/               # Alert management
  sentiment/            # Sentiment analysis
  reddit/               # Reddit trending
lib/                    # Frontend shared code
  api/                  # API client modules (one file per domain)
  auth/                 # AuthContext (JWT + localStorage)
  contexts/             # ThemeContext, ToastProvider
  hooks/                # Custom React hooks
  types/                # Frontend type definitions
  utils/                # Formatting and utility functions
types/                  # Shared TypeScript type definitions
backend/                # Go REST API
  auth/                 # JWT, middleware, rate limiting
  database/             # sqlx database layer (one file per domain)
  handlers/             # HTTP handlers (one file per domain)
  services/             # Business logic and external API clients
  migrations/           # SQL migration files (numbered)
  models/               # Go struct models
scripts/                # Python utility scripts
  us_tickers/           # Ticker data import package (tested)
migrations/             # Additional migration files
k8s/                    # Kubernetes deployment manifests
terraform/              # AWS infrastructure-as-code
.github/workflows/      # CI/CD (ci.yml, deploy-frontend.yml, deploy-backend.yml)
```

## Tech Stack

**Frontend:** Next.js 14 (App Router), React 18, TypeScript 5, Tailwind CSS 3.3
**Backend:** Go 1.21+ with Gin 1.9, sqlx (PostgreSQL), Redis, JWT auth
**Python Services:** FastAPI, Pydantic 2, psycopg2, pandas
**Charts:** Chart.js, Recharts, D3
**Deployment:** Docker, Kubernetes (AWS EKS), GitHub Actions

## Development Commands

```bash
# Frontend
npm install              # Install frontend dependencies
npm run dev              # Start Next.js dev server (localhost:3000)
npm run build            # Production build
npm run lint             # ESLint

# Backend (Go)
cd backend && go build -o investorcenter-api .   # Build
cd backend && go test ./...                       # Run all tests
cd backend && go vet ./...                        # Lint
cd backend && go fmt ./...                        # Format

# Task Service
cd task-service && go build -o task-service .     # Build
cd task-service && go test ./...                  # Test

# Python
pytest scripts/us_tickers/tests/ -v              # Run Python tests
flake8 scripts/us_tickers/ --max-line-length=79  # Lint
mypy scripts/us_tickers/                         # Type check
black scripts/us_tickers/ --line-length=79       # Format
isort scripts/us_tickers/ --line-length=79       # Sort imports

# Full project (via Makefile)
make build               # Build all (backend + task-service + frontend)
make test                # Run all tests (Python + Go)
make check               # Format + lint + test (run before pushing)
make format              # Format all code
make lint                # Lint all code
make clean               # Remove build artifacts
```

## Code Style & Conventions

### TypeScript/React
- **Path alias:** `@/*` maps to project root (e.g., `@/components/Header`, `@/lib/api/client`)
- **Components:** Feature-grouped in `components/` with `ui/` for shared primitives
- **State management:** React Context API (AuthContext, ThemeContext, ToastProvider) + localStorage for persistence. No Redux or Zustand.
- **API calls:** Use `apiClient` from `@/lib/api/client` which handles JWT auth and automatic token refresh
- **Styling:** Tailwind CSS exclusively with `ic-*` design token classes (e.g., `bg-ic-bg-primary`, `text-ic-text-secondary`). Dark mode via `[data-theme="dark"]` selector.
- **Theming colors:** Always use `ic-*` prefixed color tokens from `tailwind.config.js` rather than raw Tailwind colors. These resolve to CSS custom properties defined in `styles/theme.css`.

### Go (Backend)
- **Web framework:** Gin with route groups in `backend/main.go`
- **Database:** sqlx with raw SQL queries (not an ORM). Connection in `backend/database/db.go`.
- **File organization:** One file per domain in `handlers/`, `database/`, `services/`
- **Auth:** JWT tokens via `auth.AuthMiddleware()`, admin routes add `auth.AdminMiddleware()`
- **API prefix:** All routes under `/api/v1/`
- **Testing:** testify assertions, test files co-located as `*_test.go`
- **Error responses:** `gin.H{"error": "message"}` format

### Python
- **Line length:** 79 characters (enforced by black, flake8, isort)
- **Formatting:** black with `--line-length=79`
- **Import sorting:** isort with `--profile=black`
- **Type checking:** mypy with `--ignore-missing-imports`
- **Security:** bandit (skip B101 assert warnings)

## Database

- **Engine:** PostgreSQL with sqlx (Go) / psycopg2 (Python)
- **Migrations:** Sequential numbered SQL files in `backend/migrations/` (001 through 035+)
- **Connection pooling:** 25 max open, 5 idle, 5-min lifetime
- **Cache:** Redis for hot data (prices, volumes)

## API Structure

All API routes are defined in `backend/main.go`. Key route groups:

- **Public:** `/api/v1/auth/*`, `/api/v1/markets/*`, `/api/v1/tickers/*`, `/api/v1/stocks/*`, `/api/v1/crypto/*`, `/api/v1/reddit/*`, `/api/v1/sentiment/*`, `/api/v1/screener/*`
- **Authenticated:** `/api/v1/user/*`, `/api/v1/watchlists/*`, `/api/v1/alerts/*`, `/api/v1/notifications/*`, `/api/v1/subscriptions/*`
- **Admin:** `/api/v1/admin/*` (requires auth + admin middleware)

Frontend API modules in `lib/api/` mirror these route groups (one file per domain).

## Environment Variables

Frontend (`.env.local`):
```
NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1
NEXT_PUBLIC_IC_SCORE_API_URL=http://localhost:8001
```

Backend (see `.env.example` and `backend/main.go`):
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE`
- `JWT_SECRET` (required, validated at startup)
- `POLYGON_API_KEY`, `FMP_API_KEY`, `COINGECKO_API_KEY`
- `REDIS_URL`, `AWS_*` credentials
- `GIN_MODE` (release/debug), `PORT` (default 8080)

## CI/CD

GitHub Actions workflows in `.github/workflows/`:

- **ci.yml:** Runs on push/PR to main/develop. Python tests (matrix 3.10-3.12), linting (flake8, mypy, black, isort), security (bandit), Go tests + vet + gofmt check, package build.
- **deploy-frontend.yml:** Builds Docker, pushes to ECR, deploys to EKS on push to main.
- **deploy-backend.yml:** Same pattern for Go backend.
- **deploy-crypto-updater.yml:** Deploys crypto data service.

## Pre-commit Hooks

Configured in `.pre-commit-config.yaml`:
- **Python:** black, isort, flake8, mypy, bandit, pytest
- **Go:** gofmt check, go test

Install with: `pre-commit install`

## Testing

```bash
# Run all tests before pushing
make check

# Individual test suites
cd backend && go test ./... -v           # Go backend tests
cd task-service && go test ./...         # Task service tests
pytest scripts/us_tickers/tests/ -v     # Python tests

# Go tests with specific package
cd backend && go test ./handlers/... -v  # Handler tests only
cd backend && go test ./services/... -v  # Service tests only
```

## Key Patterns for AI Assistants

1. **Frontend API calls** always go through `lib/api/client.ts` which handles auth tokens automatically. Create new API modules in `lib/api/` following existing patterns (e.g., `watchlist.ts`, `alerts.ts`).

2. **New backend endpoints**: Add handler in `backend/handlers/`, database queries in `backend/database/`, register route in `backend/main.go`. Follow the existing pattern of one file per domain.

3. **New React components**: Place in the appropriate feature directory under `components/`. Use `ic-*` Tailwind classes for theming. Client components need `'use client'` directive.

4. **Database changes**: Add a new numbered migration file in `backend/migrations/` following the existing numbering sequence.

5. **The frontend proxies API calls** to `localhost:8080` in development via Next.js rewrites in `next.config.js`. In production, ingress handles routing.

6. **Auth flow**: JWT tokens stored in localStorage. `AuthContext` provides `user`, `login()`, `logout()`, `isAuthenticated`. Protected routes wrap with `ProtectedRoute` component.

7. **Theme system**: Dark/light/system via `ThemeContext`. Uses `data-theme` attribute on `<html>`. All colors should use `ic-*` CSS variable tokens, not raw color values.
