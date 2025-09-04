# InvestorCenter.ai

A professional financial data and analytics platform similar to YCharts, built with Next.js and deployed on AWS EKS.

## Features

- **Modern Frontend**: Next.js 14 with React 18, TypeScript, and Tailwind CSS
- **High-Performance Backend**: Go API server with Gin framework  
- **Financial Database**: PostgreSQL with 4,600+ US stocks and 9 financial data tables
- **Real-time Data**: Live market data and financial analytics
- **Professional UI**: Clean, responsive design similar to YCharts
- **Cloud-Native**: Containerized with Docker and deployed on AWS EKS

## Quick Start

```bash
# Complete setup in one command
git clone <repository-url> investorcenter.ai
cd investorcenter.ai

# Setup API keys (see docs/API_KEY_MANAGEMENT.md for details)
cp .env.example .env
# Edit .env and add your API keys

make setup

# Start development
make dev
```

This will:
1. Install all dependencies (Node.js, Go, Python, PostgreSQL)
2. Setup database with migrations and import 4,600+ stock tickers
3. Configure development environment
4. Start both backend (port 8080) and frontend (port 3000)

## Development Commands

```bash
make dev            # Start full development environment
make test           # Run all tests and linting  
make build          # Build backend and frontend
make check          # Validate code quality before push
make db-import      # Update stock ticker data
```

See `make help` for all available commands.

## Architecture

- **Frontend**: Next.js with TypeScript and Tailwind CSS
- **Backend**: Go API with Gin framework and PostgreSQL
- **Database**: PostgreSQL with financial data tables and stock tickers
- **Deployment**: Kubernetes on AWS EKS with Terraform

## Documentation

- **[SETUP.md](SETUP.md)** - Local development setup and workflow
- **[DEPLOYMENT.md](DEPLOYMENT.md)** - AWS EKS infrastructure deployment
- **[PRODUCTION_DEPLOYMENT.md](PRODUCTION_DEPLOYMENT.md)** - Production database and CronJob setup

## API Endpoints

- **GET /api/v1/tickers** - List stocks with pagination and search
- **GET /api/v1/tickers/:symbol** - Stock overview and fundamentals
- **GET /api/v1/tickers/:symbol/chart** - Historical price data
- **GET /api/v1/market/indices** - Market indices overview
- **GET /health** - Application health status

## Database Schema

9 comprehensive financial tables:
- **stocks** - Company information and metadata
- **stock_prices** - Historical OHLCV price data
- **fundamentals** - Financial metrics (PE, ROE, revenue, etc.)
- **earnings** - Quarterly earnings data
- **dividends** - Dividend payment history
- **insider_trades** - Insider trading activity
- **analyst_ratings** - Analyst recommendations
- **financial_statements** - Income statements, balance sheets
- **market_data** - Real-time market information

## Tech Stack

### Frontend
- Next.js 14 with App Router
- React 18 with TypeScript
- Tailwind CSS for styling
- Recharts for financial charts

### Backend  
- Go 1.19+ with Gin web framework
- PostgreSQL 15 for data persistence
- Redis for caching (production)
- Docker containerization

### Infrastructure
- AWS EKS for Kubernetes orchestration
- Terraform for infrastructure as code
- GitHub Actions for CI/CD
- Route53 for DNS and SSL certificates

## Contributing

1. Run `make check` before committing to ensure code quality
2. All code is automatically formatted and linted
3. Tests are required for new functionality
4. Database changes require migrations

## License

MIT License - see LICENSE file for details.