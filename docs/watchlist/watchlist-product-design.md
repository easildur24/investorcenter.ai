# Ticker Watch List System - Product Design

## Executive Summary

A personalized watch list system that allows authenticated users to track selected stocks/crypto, visualize them in custom heatmaps, and receive real-time notifications for significant events (news, earnings, price movements).

## Core Features

### 1. Account Management System

#### 1.1 Authentication & Authorization
- **Sign Up / Login**
  - Email + Password authentication
  - OAuth providers (Google, GitHub) for quick onboarding
  - Email verification for new accounts
  - Password reset flow

- **Session Management**
  - JWT-based authentication tokens
  - Refresh token rotation for security
  - Session expiry: 7 days (configurable)

- **User Profile**
  - Basic info: name, email, timezone
  - Notification preferences
  - Display preferences (theme, default view)

#### 1.2 Authorization Levels
- **Free Tier**
  - Max 10 tickers in watch list
  - Basic price alerts
  - Daily email digest

### 2. Watch List Management

#### 2.1 Core Functionality
- **Add/Remove Tickers**
  - Search and add any stock/crypto from database
  - Bulk add from CSV import
  - Quick add from any ticker page
  - Remove individual or bulk delete

- **Organization**
  - Multiple watch lists per user (e.g., "Tech Stocks", "Crypto", "Day Trade")
  - Default watch list for new users
  - Rename, reorder, delete lists
  - Share read-only watch lists via public link (future)

- **Ticker Metadata**
  - User-added notes per ticker
  - Custom tags/labels
  - Target buy/sell prices
  - Date added for tracking

#### 2.2 Watch List Views
- **Table View**
  - Sortable columns: price, change %, volume, market cap
  - Real-time price updates
  - Color-coded gains/losses

- **Heatmap View**
  - Reddit heatmap-style visualization
  - Size = Market cap or volume
  - Color = Price change % (customizable timeframe: 1D, 1W, 1M, 3M, YTD, 1Y)
  - Hover tooltip shows details

- **Chart Grid View**
  - Mini charts in grid layout
  - Sparklines showing price trend
  - Quick comparison across watch list

### 3. Custom Heatmap Generation

#### 3.1 Heatmap Configuration
- **Metric Selection**
  - Size dimension: Market cap, Volume, Avg volume, Reddit Popularity
  - Color dimension: Price change %, Volume change %, Mention 
  - Time period: 1D, 1W, 1M, 3M, 6M, YTD, 1Y, 5Y

- **Filtering**
  - Asset type: Stocks only, Crypto only, Both
  - Sector/Industry filter
  - Price range filter
  - Market cap range filter

- **Visual Customization**
  - Color scheme: Red-Green (default), Heatmap, Custom gradient
  - Label display: Symbol only, Symbol + Change, Full info
  - Grid vs Treemap layout

#### 3.2 Heatmap Features
- **Interactive**
  - Click to navigate to ticker detail page
  - Hover for detailed tooltip
  - Zoom/pan for large watch lists

- **Export**
  - Save as PNG/PDF
  - Share via unique URL
  - Embed code for external sites (future)

### 4. Notification System

#### 4.1 Alert Types

**Price Alerts**
- Price crosses above/below threshold
- Price change % in timeframe (e.g., +5% in 1 hour)
- 52-week high/low reached
- Volume spike (e.g., 3x average volume)

**News Alerts**
- Breaking news for watched tickers
- Sentiment-based filtering (positive/negative/neutral)
- Source filtering (trusted sources only)
- Keyword matching (user-defined)

**Financial Event Alerts**
- Earnings report release (pre-market)
- SEC filing published (10-K, 10-Q, 8-K, etc.)
- Dividend announcement/ex-date
- Stock split announcement

**Technical Alerts** (Premium)
- Moving average crossover (e.g., 50-day crosses 200-day)
- RSI overbought/oversold
- Bollinger Band breakout
- Custom indicator thresholds

#### 4.2 Notification Channels

**In-App Notifications**
- Bell icon in header with unread count
- Notification center with history
- Mark as read/unread
- Archive/delete

**Email Notifications**
- Immediate alerts for critical events
- Daily digest at user-configured time
- Weekly summary
- HTML formatted with charts/links

**Push Notifications** (future)
- Browser push (PWA)
- Mobile app push (iOS/Android)

**Webhook/API** (future)
- POST to user-defined URL
- Integration with Zapier/IFTC
- Slack/Discord/Telegram bots

#### 4.3 Alert Management
- **Alert Rules**
  - Create/edit/delete alert rules
  - Enable/disable without deleting
  - Set expiration date (auto-disable after date)
  - Cooldown period (don't re-alert for X hours)

- **Alert History**
  - Log of all triggered alerts
  - Search and filter history
  - Export alert history as CSV

- **Notification Preferences**
  - Quiet hours (no alerts during sleep time)
  - Channel preferences per alert type
  - Alert priority levels (critical, high, normal, low)
  - Batch similar alerts (e.g., group news from same ticker)

---

## Technical Architecture

### Database Schema

```sql
-- Users and Authentication
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255), -- NULL for OAuth users
    full_name VARCHAR(255),
    timezone VARCHAR(50) DEFAULT 'UTC',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP,
    email_verified BOOLEAN DEFAULT FALSE,
    is_premium BOOLEAN DEFAULT FALSE
);

CREATE TABLE oauth_providers (
    id SERIAL PRIMARY KEY,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL, -- 'google', 'github'
    provider_user_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(provider, provider_user_id)
);

CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    refresh_token_hash VARCHAR(255),
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Watch Lists
CREATE TABLE watch_lists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_default BOOLEAN DEFAULT FALSE,
    display_order INTEGER,
    is_public BOOLEAN DEFAULT FALSE, -- For sharing feature
    public_slug VARCHAR(100) UNIQUE, -- For public URLs
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE watch_list_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    watch_list_id UUID REFERENCES watch_lists(id) ON DELETE CASCADE,
    symbol VARCHAR(20) NOT NULL, -- References stocks.symbol
    notes TEXT,
    tags TEXT[], -- Array of custom tags
    target_buy_price DECIMAL(20, 4),
    target_sell_price DECIMAL(20, 4),
    added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    display_order INTEGER,
    UNIQUE(watch_list_id, symbol)
);

CREATE INDEX idx_watch_list_items_list_id ON watch_list_items(watch_list_id);
CREATE INDEX idx_watch_list_items_symbol ON watch_list_items(symbol);
CREATE INDEX idx_watch_lists_user_id ON watch_lists(user_id);

-- Alert System
CREATE TABLE alert_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    watch_list_id UUID REFERENCES watch_lists(id) ON DELETE CASCADE, -- NULL = applies to all watch lists
    symbol VARCHAR(20), -- NULL = applies to all tickers in watch list
    alert_type VARCHAR(50) NOT NULL, -- 'price_above', 'price_below', 'price_change_pct', 'volume_spike', 'news', 'earnings', 'sec_filing'
    condition_json JSONB NOT NULL, -- Flexible conditions: {"threshold": 100, "operator": ">=", "timeframe": "1h"}
    is_enabled BOOLEAN DEFAULT TRUE,
    expires_at TIMESTAMP, -- Auto-disable after this date
    cooldown_minutes INTEGER DEFAULT 60, -- Don't re-alert for X minutes
    last_triggered_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE alert_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_rule_id UUID REFERENCES alert_rules(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    symbol VARCHAR(20) NOT NULL,
    alert_type VARCHAR(50) NOT NULL,
    triggered_value DECIMAL(20, 4), -- The value that triggered the alert
    message TEXT NOT NULL,
    metadata_json JSONB, -- Additional context
    triggered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    read_at TIMESTAMP,
    archived_at TIMESTAMP
);

CREATE INDEX idx_alert_logs_user_id_triggered_at ON alert_logs(user_id, triggered_at DESC);
CREATE INDEX idx_alert_logs_symbol ON alert_logs(symbol);
CREATE INDEX idx_alert_rules_user_id ON alert_rules(user_id);

-- Notification Preferences
CREATE TABLE notification_preferences (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    email_enabled BOOLEAN DEFAULT TRUE,
    email_immediate_enabled BOOLEAN DEFAULT TRUE,
    email_digest_enabled BOOLEAN DEFAULT TRUE,
    email_digest_time TIME DEFAULT '09:00:00', -- Local time based on user.timezone
    push_enabled BOOLEAN DEFAULT FALSE,
    in_app_enabled BOOLEAN DEFAULT TRUE,
    quiet_hours_start TIME, -- e.g., '22:00:00'
    quiet_hours_end TIME, -- e.g., '07:00:00'
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Heatmap Configurations (Save user's custom heatmap settings)
CREATE TABLE heatmap_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    watch_list_id UUID REFERENCES watch_lists(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    size_metric VARCHAR(50) DEFAULT 'market_cap', -- 'market_cap', 'volume', 'avg_volume'
    color_metric VARCHAR(50) DEFAULT 'price_change_pct', -- 'price_change_pct', 'volume_change_pct'
    time_period VARCHAR(10) DEFAULT '1D', -- '1D', '1W', '1M', '3M', '6M', 'YTD', '1Y', '5Y'
    color_scheme VARCHAR(50) DEFAULT 'red_green', -- 'red_green', 'heatmap', 'custom'
    filters_json JSONB, -- Asset type, sector, price range, etc.
    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Backend Architecture

#### New Go Packages

```
backend/
├── auth/
│   ├── jwt.go                 # JWT token generation and validation
│   ├── password.go            # Password hashing with bcrypt
│   ├── oauth.go               # OAuth provider integrations
│   └── middleware.go          # Auth middleware for protected routes
│
├── handlers/
│   ├── auth_handlers.go       # Sign up, login, logout, password reset
│   ├── user_handlers.go       # User profile, preferences
│   ├── watchlist_handlers.go  # CRUD for watch lists and items
│   ├── alert_handlers.go      # CRUD for alert rules, fetch alert logs
│   ├── notification_handlers.go  # Notification preferences, mark as read
│   └── heatmap_handlers.go    # Generate heatmap data for watch list
│
├── services/
│   ├── auth_service.go        # Authentication business logic
│   ├── user_service.go        # User management
│   ├── watchlist_service.go   # Watch list operations
│   ├── alert_service.go       # Alert rule management
│   ├── alert_processor.go     # Background worker to evaluate alert rules
│   ├── notification_service.go  # Send notifications (email, push, in-app)
│   └── heatmap_service.go     # Generate heatmap data from watch list
│
├── database/
│   ├── users.go               # User CRUD
│   ├── watchlists.go          # Watch list CRUD
│   ├── alerts.go              # Alert rules and logs CRUD
│   └── notifications.go       # Notification preferences CRUD
│
├── models/
│   ├── user.go                # User, Session, OAuthProvider
│   ├── watchlist.go           # WatchList, WatchListItem
│   ├── alert.go               # AlertRule, AlertLog
│   ├── notification.go        # NotificationPreference
│   └── heatmap.go             # HeatmapConfig, HeatmapData
│
└── workers/
    ├── alert_worker.go        # Periodic job to check alert conditions
    ├── email_worker.go        # Queue-based email sending
    └── digest_worker.go       # Daily/weekly digest generation
```

#### API Endpoints

```
# Authentication
POST   /api/v1/auth/signup              # Create new account
POST   /api/v1/auth/login               # Email/password login
POST   /api/v1/auth/logout              # Invalidate session
POST   /api/v1/auth/refresh             # Refresh JWT token
POST   /api/v1/auth/forgot-password     # Send password reset email
POST   /api/v1/auth/reset-password      # Reset password with token
GET    /api/v1/auth/verify-email/:token # Verify email address
POST   /api/v1/auth/oauth/:provider     # OAuth login (Google, GitHub)

# User Profile
GET    /api/v1/user/me                  # Get current user info
PUT    /api/v1/user/me                  # Update profile
PUT    /api/v1/user/password            # Change password
DELETE /api/v1/user/me                  # Delete account

# Watch Lists
GET    /api/v1/watchlists               # List all user's watch lists
POST   /api/v1/watchlists               # Create new watch list
GET    /api/v1/watchlists/:id           # Get watch list details with items
PUT    /api/v1/watchlists/:id           # Update watch list metadata
DELETE /api/v1/watchlists/:id           # Delete watch list
POST   /api/v1/watchlists/:id/items     # Add ticker to watch list
DELETE /api/v1/watchlists/:id/items/:symbol  # Remove ticker from watch list
PUT    /api/v1/watchlists/:id/items/:symbol  # Update ticker notes/tags/targets
POST   /api/v1/watchlists/:id/bulk      # Bulk add tickers (CSV import)
GET    /api/v1/watchlists/:id/heatmap   # Get heatmap data for watch list

# Alerts
GET    /api/v1/alerts/rules             # List all alert rules
POST   /api/v1/alerts/rules             # Create new alert rule
GET    /api/v1/alerts/rules/:id         # Get alert rule details
PUT    /api/v1/alerts/rules/:id         # Update alert rule
DELETE /api/v1/alerts/rules/:id         # Delete alert rule
POST   /api/v1/alerts/rules/:id/toggle  # Enable/disable alert rule

GET    /api/v1/alerts/logs              # Get alert logs (paginated, filterable)
GET    /api/v1/alerts/logs/:id          # Get specific alert log
PUT    /api/v1/alerts/logs/:id/read     # Mark alert as read
PUT    /api/v1/alerts/logs/bulk-read    # Mark multiple alerts as read
DELETE /api/v1/alerts/logs/:id          # Delete/archive alert log

# Notifications
GET    /api/v1/notifications/preferences   # Get notification preferences
PUT    /api/v1/notifications/preferences   # Update notification preferences
GET    /api/v1/notifications/unread-count  # Get count of unread alerts

# Heatmap Configs (Save custom heatmap settings)
GET    /api/v1/heatmaps                 # List saved heatmap configs
POST   /api/v1/heatmaps                 # Save new heatmap config
GET    /api/v1/heatmaps/:id             # Get heatmap config
PUT    /api/v1/heatmaps/:id             # Update heatmap config
DELETE /api/v1/heatmaps/:id             # Delete heatmap config
```

### Frontend Architecture

#### New Pages and Components

```
app/
├── auth/
│   ├── login/page.tsx         # Login page
│   ├── signup/page.tsx        # Sign up page
│   ├── forgot-password/page.tsx   # Password reset request
│   └── reset-password/page.tsx    # Password reset with token
│
├── watchlist/
│   ├── page.tsx               # Watch list dashboard (list all watch lists)
│   ├── [id]/page.tsx          # Single watch list view (table/heatmap)
│   └── [id]/heatmap/page.tsx  # Full-screen heatmap view
│
├── alerts/
│   ├── page.tsx               # Alert rules management
│   └── logs/page.tsx          # Alert history/logs
│
├── settings/
│   ├── profile/page.tsx       # User profile settings
│   ├── notifications/page.tsx # Notification preferences
│   └── account/page.tsx       # Account settings, delete account
│
└── layout.tsx                 # Update with auth context and notification bell

components/
├── auth/
│   ├── LoginForm.tsx          # Login form component
│   ├── SignUpForm.tsx         # Sign up form component
│   ├── AuthProvider.tsx       # React Context for auth state
│   └── ProtectedRoute.tsx     # HOC for protected pages
│
├── watchlist/
│   ├── WatchListCard.tsx      # Watch list preview card
│   ├── WatchListTable.tsx     # Table view of watch list items
│   ├── WatchListHeatmap.tsx   # Heatmap visualization (D3.js or recharts)
│   ├── AddTickerModal.tsx     # Modal to add ticker to watch list
│   ├── EditTickerModal.tsx    # Modal to edit ticker notes/targets
│   └── CreateWatchListModal.tsx  # Modal to create new watch list
│
├── alerts/
│   ├── AlertRuleForm.tsx      # Form to create/edit alert rule
│   ├── AlertRuleCard.tsx      # Display alert rule with toggle
│   ├── AlertLogItem.tsx       # Single alert log entry
│   └── NotificationBell.tsx   # Header bell icon with unread count
│
└── settings/
    ├── ProfileForm.tsx        # User profile edit form
    ├── NotificationPreferencesForm.tsx  # Notification settings
    └── QuietHoursSelector.tsx # Time range picker for quiet hours

lib/
├── auth.ts                    # Auth API calls, token management
├── watchlist.ts               # Watch list API calls
├── alerts.ts                  # Alert API calls
└── hooks/
    ├── useAuth.tsx            # Hook to access auth context
    ├── useWatchLists.ts       # Hook to fetch/manage watch lists
    ├── useAlerts.ts           # Hook to fetch/manage alerts
    └── useNotifications.ts    # Hook for notification bell
```

---

## Implementation Phases

### Phase 1: Authentication & User Management (Week 1-2)
**Goal:** Users can sign up, login, and manage their profile

- [ ] Database migrations for users, sessions, oauth_providers
- [ ] Backend: JWT auth, password hashing, session management
- [ ] Backend: Auth middleware for protected routes
- [ ] Backend: User CRUD endpoints
- [ ] Frontend: Login, signup, forgot password pages
- [ ] Frontend: Auth context and protected route HOC
- [ ] Frontend: Update header with user menu (profile, logout)
- [ ] Testing: Auth flows, token refresh, session expiry

**Deliverable:** Working authentication system, users can create accounts and login

---

### Phase 2: Watch List Management (Week 3-4)
**Goal:** Users can create watch lists and add tickers

- [ ] Database migrations for watch_lists, watch_list_items
- [ ] Backend: Watch list CRUD endpoints
- [ ] Backend: Add/remove tickers from watch list
- [ ] Backend: Bulk import from CSV
- [ ] Frontend: Watch list dashboard page
- [ ] Frontend: Create/edit/delete watch list modals
- [ ] Frontend: Add/remove tickers (search autocomplete)
- [ ] Frontend: Table view with real-time prices
- [ ] Frontend: Ticker notes, tags, target prices UI
- [ ] Testing: Watch list operations, bulk import, max ticker limits

**Deliverable:** Fully functional watch list management, users can organize their tickers

---

### Phase 3: Custom Heatmap Visualization (Week 5-6)
**Goal:** Users can visualize their watch lists as interactive heatmaps

- [ ] Database migrations for heatmap_configs
- [ ] Backend: Heatmap data generation endpoint
- [ ] Backend: Fetch real-time prices for watch list tickers
- [ ] Backend: Calculate metrics (market cap, volume, change %)
- [ ] Backend: Save/load heatmap configurations
- [ ] Frontend: Heatmap component (D3.js treemap or recharts)
- [ ] Frontend: Heatmap configuration panel (metrics, timeframe, colors)
- [ ] Frontend: Interactive hover tooltips
- [ ] Frontend: Click to navigate to ticker detail page
- [ ] Frontend: Save/load custom heatmap configs
- [ ] Testing: Heatmap rendering performance, data accuracy

**Deliverable:** Interactive heatmap for watch lists, similar to Reddit heatmap

---

### Phase 4: Alert System - Price & Volume (Week 7-8)
**Goal:** Users can set price and volume alerts

- [ ] Database migrations for alert_rules, alert_logs
- [ ] Backend: Alert rule CRUD endpoints
- [ ] Backend: Alert log endpoints (list, mark as read)
- [ ] Backend: Alert processor worker (checks conditions periodically)
- [ ] Backend: In-app notification creation
- [ ] Backend: Email notification service (SMTP or SendGrid)
- [ ] Frontend: Alert rules management page
- [ ] Frontend: Create/edit alert rule form
- [ ] Frontend: Alert rule cards with enable/disable toggle
- [ ] Frontend: Alert logs page (notification center)
- [ ] Frontend: Notification bell in header with unread count
- [ ] Frontend: Mark as read/unread, delete alerts
- [ ] Testing: Alert triggering logic, cooldown periods, notification delivery

**Deliverable:** Working price and volume alert system with email notifications

---

### Phase 5: Alert System - News & Financial Events (Week 9-10)
**Goal:** Users get notified about news and earnings for watched tickers

- [ ] Backend: Integrate news API (Polygon.io news or NewsAPI)
- [ ] Backend: Fetch SEC filings for watched tickers (use existing sec_filings table)
- [ ] Backend: Earnings calendar integration (Polygon.io or Alpha Vantage)
- [ ] Backend: News alert processor (check for new articles)
- [ ] Backend: SEC filing alert processor (check for new filings)
- [ ] Backend: Earnings alert processor (check for upcoming earnings)
- [ ] Backend: Sentiment analysis for news (optional, use external API)
- [ ] Frontend: News alert rule configuration
- [ ] Frontend: Financial event alert rule configuration
- [ ] Frontend: Alert logs with news article links
- [ ] Frontend: Alert logs with SEC filing links
- [ ] Testing: News fetching, filing detection, earnings calendar accuracy

**Deliverable:** Comprehensive alert system covering all major event types

---

### Phase 6: Notification Preferences & Digest (Week 11-12)
**Goal:** Users can customize notification channels and receive daily digests

- [ ] Database migrations for notification_preferences
- [ ] Backend: Notification preferences endpoints
- [ ] Backend: Quiet hours logic in alert processor
- [ ] Backend: Daily digest worker (aggregate alerts, send at scheduled time)
- [ ] Backend: Weekly summary worker
- [ ] Backend: Email templates (HTML) for alerts and digests
- [ ] Frontend: Notification preferences page
- [ ] Frontend: Quiet hours time picker
- [ ] Frontend: Email digest time selector
- [ ] Frontend: Channel toggles (email immediate, digest, in-app)
- [ ] Testing: Quiet hours enforcement, digest scheduling, timezone handling

**Deliverable:** Full notification customization, daily/weekly email digests

---

### Phase 7: Polish & Premium Features (Week 13-14)
**Goal:** Refine UX, add premium features, prepare for launch

- [ ] Backend: User tier management (free vs premium)
- [ ] Backend: Enforce watch list limits for free tier (max 10 tickers)
- [ ] Backend: Premium-only alert types (technical indicators)
- [ ] Backend: Public watch list sharing (generate public URLs)
- [ ] Frontend: Upgrade to premium CTA
- [ ] Frontend: Public watch list view (read-only)
- [ ] Frontend: Heatmap export as PNG
- [ ] Frontend: Mobile responsive design improvements
- [ ] Frontend: Loading states, error handling, empty states
- [ ] Frontend: Onboarding tour for new users
- [ ] Testing: End-to-end user flows, edge cases
- [ ] Documentation: API docs, user guide

**Deliverable:** Production-ready watch list system with premium tier

---

## Technical Considerations

### Performance Optimization

1. **Real-time Price Updates**
   - Use WebSocket connection for watch list page to push price updates
   - Fallback to polling every 5 seconds if WebSocket unavailable
   - Batch database queries for watch list items (fetch all prices in one query)

2. **Alert Processing**
   - Run alert worker as Kubernetes CronJob every 1 minute
   - Process alerts in batches (e.g., 1000 users per run)
   - Use Redis queue for alert notifications to avoid email bottlenecks
   - Cache alert rule conditions in Redis to avoid database hits

3. **Heatmap Generation**
   - Pre-calculate heatmap data for default configurations
   - Cache heatmap data in Redis for 5 minutes
   - Use database indexes on watch_list_items.symbol for fast joins
   - Limit max tickers in heatmap to 200 for rendering performance

4. **Database Optimization**
   - Add indexes on foreign keys, user_id, symbol, triggered_at
   - Partition alert_logs table by month if it grows large
   - Use database connection pooling (already in place)
   - Archive old alert logs after 90 days

### Security Considerations

1. **Authentication**
   - Use bcrypt for password hashing (cost factor 12)
   - JWT tokens with short expiry (1 hour), refresh tokens (7 days)
   - Store refresh tokens as hashed values in database
   - Implement rate limiting on login endpoint (5 attempts per 15 min)

2. **Authorization**
   - Verify user owns watch list before allowing access
   - Verify user owns alert rule before allowing access
   - Use prepared statements to prevent SQL injection
   - Validate all user inputs (email format, password strength, etc.)

3. **Data Privacy**
   - Don't expose user emails in API responses
   - Allow users to delete their account (cascade delete all data)
   - Encrypt sensitive data in database (e.g., OAuth tokens)
   - Comply with GDPR/CCPA (data export, deletion requests)

4. **Rate Limiting**
   - Limit API calls per user (100 requests per minute)
   - Limit alert rule creation (max 50 rules per user)
   - Limit email notifications (max 100 emails per day per user)
   - Use Redis for distributed rate limiting across instances

### Monitoring & Observability

1. **Metrics to Track**
   - User signups per day
   - Active users (daily, weekly, monthly)
   - Average watch list size
   - Alert rules created per user
   - Alert trigger rate (alerts triggered per day)
   - Email delivery success rate
   - API response times (p50, p95, p99)
   - Database query performance

2. **Logging**
   - Log all authentication events (login, logout, failed attempts)
   - Log alert triggers with rule ID and user ID
   - Log email sending (success/failure)
   - Log API errors with stack traces
   - Use structured logging (JSON format)

3. **Alerting** (for us, not users)
   - Alert if API error rate > 5%
   - Alert if database connection fails
   - Alert if email delivery rate < 90%
   - Alert if alert worker stops running
   - Alert if disk space < 20%

---

## Open Questions & Future Enhancements

### Open Questions
1. **Should we allow anonymous watch lists?** (No login required, store in localStorage)
   - Pro: Lower barrier to entry, faster onboarding
   - Con: Can't sync across devices, lose data if clear browser

2. **What OAuth providers to support initially?**
   - Recommendation: Start with Google (most common), add GitHub later

3. **How to handle deleted tickers?**
   - If a ticker is delisted, should we soft-delete or show as "Delisted"?

4. **Should watch lists be private by default or public?**
   - Recommendation: Private by default, opt-in to public sharing

5. **What's the pricing for premium tier?**
   - Need to define value proposition and competitive pricing

### Future Enhancements

**Social Features**
- Follow other users' public watch lists
- Leaderboard for best-performing watch lists
- Comments and discussions on public watch lists
- Social sharing (share watch list on Twitter, Reddit)

**Advanced Analytics**
- Watch list performance tracking (total gain/loss, best/worst performers)
- Portfolio allocation pie chart (sector breakdown)
- Correlation matrix between tickers in watch list
- Backtesting: "What if I bought all these tickers 1 year ago?"

**Mobile App**
- React Native app for iOS and Android
- Push notifications for alerts
- Widget on home screen showing watch list summary

**AI-Powered Features**
- AI-generated watch list suggestions based on user's interests
- Sentiment analysis on news for watched tickers
- Predictive alerts (e.g., "X:BTCUSD may drop 10% based on patterns")
- Natural language alert creation ("Alert me when AAPL crosses $200")

**Integration with Brokers**
- Import portfolio from Robinhood, Fidelity, etc.
- Execute trades directly from InvestorCenter.ai
- Sync watch list with broker's watch list

**Collaboration**
- Team watch lists (shared editing, multiple users)
- Watch list templates (e.g., "Tech Giants", "Dividend Stocks")
- Export watch list to Excel/Google Sheets

---

## Success Metrics

### Launch Goals (3 months post-launch)
- 1,000 registered users
- 500 active users (login in last 7 days)
- 5,000 watch lists created
- 10,000 alert rules created
- 1,000 alerts triggered per day
- 50% user retention (users who return after first visit)

### Product-Market Fit Indicators
- Users create > 1 watch list on average
- Users add > 5 tickers to watch lists
- Users create > 2 alert rules
- Users login > 2 times per week
- Email open rate > 30% for alert notifications
- NPS score > 40

---

## Conclusion

This watch list system transforms InvestorCenter.ai from a data viewing platform into a personalized financial monitoring hub. By combining custom heatmap visualizations with intelligent alerts, we give users powerful tools to stay informed about their investments.

The phased approach allows us to iterate quickly and gather user feedback early. Starting with core authentication and watch list features (Phases 1-2) provides immediate value, while alert system (Phases 4-5) and notification customization (Phase 6) build on that foundation to create a sticky, habit-forming product.

The architecture is designed to scale, with background workers for alert processing, Redis caching for performance, and a flexible alert rule system that can accommodate future alert types without schema changes.

**Next Steps:**
1. Review and refine this design with the team
2. Create detailed technical specs for Phase 1 (Auth)
3. Set up project tracking (GitHub Projects or Jira)
4. Begin implementation!
