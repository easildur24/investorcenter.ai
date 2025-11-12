# Implementation Prompts: Phase 2 & 3
## InvestorCenter.ai - Months 7-18

**Version:** 1.0
**Date:** November 12, 2025

---

## Phase 2: Differentiation (Months 7-12)

### Prompt 1: Analyst Track Record System

```
Implement an analyst tracking and ranking system similar to TipRanks, tracking
performance of 10,000+ Wall Street analysts.

Requirements:
- Track every analyst recommendation with outcome
- Calculate success rate, average return, recommendation count
- Assign 1-5 star ratings based on performance
- Filter consensus by top analysts only
- Analyst leaderboard and search
- Historical accuracy over 1Y, 3Y, 5Y periods

Technical:
- Database schema for analyst_performance table
- Background job to calculate performance metrics daily
- API endpoints for analyst data
- Frontend components for analyst rankings display

Data needed:
- Historical analyst ratings (collect prospectively)
- Price data to measure recommendation outcomes
- Time-weighted return calculations
```

### Prompt 2: Social Media Sentiment Analysis

```
Build social sentiment tracking for Reddit, Twitter/X, and StockTwits to enhance
our existing Reddit heatmap data.

Requirements:
- Reddit API integration (we have existing infrastructure!)
- Twitter/X API integration (mentions, sentiment)
- StockTwits API integration
- NLP sentiment analysis for each platform
- Aggregate "Social Sentiment Score" 0-100
- Trending stocks detection
- Integration into IC Score (leverage existing sentiment factor)

Technical:
- Reddit: Expand existing scraping to include r/stocks, r/investing
- Twitter: Stream API for real-time mentions
- StockTwits: REST API for stock-specific sentiment
- FinBERT or custom model for financial sentiment
- Aggregate metrics: mention volume, sentiment distribution
- Display: Social sentiment dashboard, trending stocks widget
```

### Prompt 3: AI-Powered Insights Generator

```
Implement AI-generated insights and summaries using LLMs (GPT-4, Claude).

Features to implement:
1. Stock Summary ("What You Need to Know")
   - Analyze IC Score factors
   - Summarize key strengths/weaknesses
   - Identify catalysts and risks
   - Investment thesis in 2-3 paragraphs

2. Earnings Summary
   - Parse earnings call transcripts
   - Extract key highlights, guidance, management tone
   - Compare to prior quarters
   - Analyst Q&A takeaways

3. News Summary
   - Digest last 7 days of news
   - Extract key developments
   - Identify conflicting viewpoints
   - Assess implications for stock

4. Portfolio Insights
   - Analyze user portfolio
   - Identify concentration risks
   - Suggest rebalancing
   - Alert to declining holdings

Technical Implementation:
- Use OpenAI GPT-4 or Anthropic Claude API
- Create prompt templates for each insight type
- Cache results (24h TTL) to control costs
- Implement streaming responses for better UX
- Fact-checking layer to prevent hallucinations
- Cost monitoring and rate limiting

Example prompt template:
"Analyze AAPL with IC Score 78/100. Factors: Value 62 (C+),
Growth 88 (A-), Profitability 95 (A+), Financial Health 85 (A),
Momentum 78 (B+). Recent events: Earnings beat by 8.5%, insider
buying $12M. Provide 3-paragraph investment summary covering
strengths, concerns, and outlook."
```

### Prompt 4: Historical IC Score Tracking & Visualization

```
Implement comprehensive historical tracking of IC Scores with interactive
visualizations and performance analysis.

Requirements:
1. Historical score storage (already in ic_scores table)
2. Interactive charts showing score evolution
3. Overlay price performance for correlation analysis
4. Factor contribution changes over time
5. Event annotations (earnings, news, analyst changes)
6. Score vs. returns backtesting
7. Stability metrics (score volatility)

Visualizations:
- Line chart: IC Score over time (1M, 3M, 6M, 1Y, 2Y, 5Y)
- Dual-axis: Score + Price on same chart
- Stacked area: Individual factor contributions
- Heatmap: All factors over time (color intensity = score)
- Scatter plot: Score change vs. Price change (correlation)

Analytics:
- "Stocks with IC Score >80 returned avg +18.5% over next 6M"
- "Score improvements of >10 points led to +12.3% avg return"
- Factor predictiveness: Which factors most correlated with future returns?

Technical:
- TimescaleDB continuous aggregates for fast queries
- D3.js or Recharts for interactive visualizations
- Annotation system for events
- Export to image functionality
```

### Prompt 5: Economic Data Integration

```
Integrate macroeconomic indicators and overlay on stock charts like YCharts.

Data Sources:
- FRED (Federal Reserve Economic Data) - FREE API
- GDP, unemployment, CPI, interest rates, etc.

Features:
1. Economic Dashboard
   - Current values for 20+ key indicators
   - Trend analysis (improving/declining)
   - Recession probability indicator

2. Chart Overlays
   - Recession shading (gray areas during NBER recessions)
   - Overlay any economic indicator on stock chart
   - Compare stock vs. economic trend
   - Correlation analysis

3. Sector Impact Analysis
   - Which sectors benefit from rising rates?
   - Tech performance during recessions
   - Cyclical vs. defensive correlation

4. Economic Calendar
   - Upcoming data releases (CPI, jobs report, GDP)
   - Fed meeting dates
   - Expected values vs. actual

Technical:
- FRED API client for data fetching
- Store economic data in time-series table
- Chart.js plugin for dual-axis charts
- Background color for recession periods
- Tooltip showing both stock and economic values
```

### Prompt 6: PDF Report Generation

```
Implement professional PDF report generation for stocks and portfolios.

Report Types:

1. Stock Analysis Report (10-15 pages)
   - Cover page with logo and stock name
   - Executive summary (1 page)
   - IC Score breakdown (2 pages)
   - Financial highlights with charts (3 pages)
   - Valuation analysis vs. peers (2 pages)
   - Technical analysis with chart (1 page)
   - Analyst consensus (1 page)
   - News summary (1 page)
   - Appendix: Methodology, disclaimers

2. Portfolio Report (15-20 pages)
   - Cover page
   - Portfolio overview (holdings table)
   - Performance metrics (charts + tables)
   - Sector allocation (pie chart)
   - Risk analysis (concentration, volatility)
   - Top/bottom performers
   - Dividend income summary
   - Recommendations
   - Appendix

Customization:
- Choose sections to include
- Custom branding (upload logo, choose colors) - Premium feature
- Date range selection
- Add custom notes/commentary
- Multiple templates (Professional, Detailed, Summary)

Technical Implementation:
- Use WeasyPrint or ReportLab (Python) for PDF generation
- HTML/CSS templates for layout
- Async job queue (Celery) for generation
- Store PDFs in S3
- Download link expires after 7 days
- Email delivery option
```

---

## Phase 3: Enterprise & Advanced (Months 13-18)

### Prompt 1: REST API for Developers

```
Build comprehensive REST API for programmatic access to InvestorCenter data.

Requirements:
- All stock data endpoints
- IC Score and factor data
- Historical data
- Portfolio analysis
- Screening
- Rate limiting by tier
- API keys management
- Usage analytics dashboard

Documentation:
- OpenAPI/Swagger auto-generated docs
- Code examples (Python, JavaScript, cURL)
- Postman collection
- SDKs (Python, JavaScript)

Rate Limits:
- Free: 100 requests/day
- Premium: 1,000 requests/day
- Pro: 10,000 requests/day
- Enterprise: Unlimited (custom SLA)

Implementation:
- Use FastAPI (already have this)
- API key authentication (in addition to JWT)
- Request/response logging
- Usage tracking per API key
- Billing integration (Stripe metered billing)
```

### Prompt 2: Excel Add-In

```
Create Microsoft Excel Add-In for pulling InvestorCenter data into spreadsheets,
similar to YCharts Excel Add-In.

Features:
- Custom formulas: =IC.Score("AAPL"), =IC.Factor("MSFT", "Growth")
- =IC.Price("GOOGL"), =IC.Financials("AMZN", "Revenue", "Q4 2024")
- Batch data fetching for efficiency
- One-click refresh all data
- Authentication from Excel
- Works on Windows and Mac

Technical:
- Office Add-In framework (JavaScript + React)
- Custom functions using Excel JavaScript API
- Batch API requests (fetch 100 cells at once)
- Caching to minimize API calls
- OAuth for authentication
- Publish to Microsoft AppSource
```

### Prompt 3: Brokerage Integration (Plaid)

```
Integrate with brokerage accounts for automatic portfolio syncing using Plaid.

Supported Brokers:
- Interactive Brokers, TD Ameritrade, E*TRADE
- Charles Schwab, Fidelity, Robinhood
- Others via Plaid

Features:
- Link brokerage account (OAuth)
- Automatic portfolio sync
- Real-time holdings updates
- Transaction history import
- Accurate cost basis
- Multi-account support
- Dividend tracking

Technical Implementation:
- Plaid Link SDK integration
- Plaid API for account access
- Webhook for transaction updates
- Reconciliation with manually entered data
- Data encryption (holdings data is sensitive)
- Compliance: SEC custody rules don't apply (read-only)

Security:
- Never store Plaid credentials
- Encrypt access tokens
- Revoke access on user request
- Audit log of all access
```

### Prompt 4: Native Mobile Apps (iOS & Android)

```
Build native mobile apps for iOS and Android.

Core Features (V1):
- Stock search and details
- IC Score display
- Watchlist management
- Portfolio tracking
- Price alerts (push notifications)
- News feed
- Basic charting
- Simplified screener

Technology:
- iOS: Swift, SwiftUI
- Android: Kotlin, Jetpack Compose
- Shared API client
- Push notifications: Firebase Cloud Messaging
- Offline caching: Room (Android), Core Data (iOS)

Unique Mobile Features:
- Home screen widgets (portfolio performance)
- Watch app (watchlist prices)
- Face ID / Touch ID login
- Share sheet integration (share stocks)
- Haptic feedback
- Dark mode

Development:
- Use same REST API as web
- WebSocket for real-time updates
- Optimized for mobile bandwidth
- App Store & Google Play distribution
```

### Prompt 5: Natural Language Queries

```
Implement natural language query interface for stock screening and analysis.

Example Queries:
- "Show me tech stocks with P/E under 20"
- "Find dividend stocks with high safety scores"
- "Which stocks have insider buying recently?"
- "Compare AAPL and MSFT"
- "What are the best undervalued growth stocks?"

Implementation:
- Use OpenAI GPT-4 for query understanding
- Convert natural language → structured filters
- Execute screener or API call
- Present results naturally
- Follow-up questions support

Example Flow:
User: "Show me tech stocks with high IC Scores"
AI: Converts to: {sector: "Technology", ic_score: {gte: 75}}
System: Executes screener
AI: "Found 32 technology stocks with IC Score ≥75. Top 5: MSFT (82), AAPL (78), ..."
User: "Filter to only those with P/E under 30"
AI: Adds filter: {pe_ratio: {lte: 30}}
System: Re-executes
AI: "Narrowed to 18 stocks. Top 5: ..."

Technical:
- LLM prompt engineering for financial domain
- Schema mapping (natural language → database fields)
- Query validation
- Conversation memory (context window)
- Cost optimization (cache common queries)
```

### Prompt 6: Community Features

```
Build community features for user-generated content and discussions.

Features:
1. Stock Discussion Boards
   - Thread per stock ticker
   - Upvote/downvote posts
   - Sort by: Hot, New, Top
   - Threaded replies

2. User-Shared Screens
   - Publish screener to community
   - Fork and modify others' screens
   - Upvote best screens
   - Categories: Value, Growth, Dividend, etc.

3. Portfolio Sharing (Optional)
   - Share portfolio publicly (anonymized)
   - Compare to other users
   - Leaderboards

4. User Profiles
   - Avatar, bio
   - Reputation points
   - Badges (Contributor, Expert, etc.)
   - Follow other users

5. Expert Contributors (Premium)
   - Verified accounts for analysts
   - Featured content
   - Premium analysis articles

Moderation:
- Community guidelines
- Report inappropriate content
- Auto-moderation (spam detection)
- Moderator dashboard
- Ban/suspend users

Technical:
- Separate database for community content
- Full-text search for posts
- Redis for caching hot discussions
- Elasticsearch for search
- Image uploads (S3)
- Markdown support for posts
- Rate limiting (prevent spam)

Legal/Compliance:
- Clearly state: Not financial advice
- User-generated content disclaimer
- DMCA compliance
- Privacy policy updates
```

---

## Implementation Priority for Phase 2 & 3

### Must-Have (Phase 2):
1. ✅ Analyst track records
2. ✅ Social media sentiment (leverage Reddit!)
3. ✅ AI-powered insights
4. ✅ Historical IC Score tracking
5. ✅ PDF reports
6. ✅ Excel export (basic)

### Should-Have (Phase 3):
1. ✅ REST API
2. ✅ Excel Add-In
3. ✅ Brokerage integration
4. ✅ Mobile apps
5. ✅ Natural language queries

### Nice-to-Have (Phase 3+):
1. ✅ Community features
2. Advanced backtesting
3. Portfolio optimization
4. Options analysis
5. International stocks

---

**End of Implementation Prompts - Phase 2 & 3**
