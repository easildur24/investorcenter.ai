# Comprehensive Test Plan: Watchlist Alert System

**Version**: 1.0
**Date**: 2025-11-04
**Status**: Ready for Testing

---

## Overview

This test plan covers the complete watchlist alert system implementation including:
- Alert CRUD operations
- Alert triggering and condition evaluation
- Notification delivery (email and in-app)
- Subscription limit enforcement
- Frontend UI components
- Performance and reliability

---

## Prerequisites

### Test Environment Setup

1. **Backend API**: Running at `http://localhost:8080` or production URL
2. **Frontend**: Running at `http://localhost:3000` or production URL
3. **Database**: PostgreSQL with all migrations applied
4. **Test User**: Create a test user with known credentials
5. **Test Data**:
   - At least one watchlist with 5+ symbols
   - Test symbols: AAPL, MSFT, GOOGL, TSLA, AMZN
6. **API Keys**: Valid Polygon.io API key configured

### Test Accounts

Create test accounts for each subscription tier:
- **Free Account**: `test-free@investorcenter.ai` (limits: 3 lists, 10 items, 10 alerts)
- **Premium Account**: `test-premium@investorcenter.ai` (limits: 20 lists, 100 items, 100 alerts)
- **Enterprise Account**: `test-enterprise@investorcenter.ai` (limits: unlimited)

---

## Test Cases

### Section 1: Alert Creation (API Tests)

#### Test 1.1: Create Price Above Alert
**Endpoint**: `POST /api/v1/alerts`
**Prerequisites**: User has a watchlist with AAPL

**Request Body**:
```json
{
  "watch_list_id": "{{watchlist_id}}",
  "symbol": "AAPL",
  "name": "AAPL Price Above $150",
  "alert_type": "price_above",
  "conditions": {
    "threshold": 150.00,
    "comparison": "above"
  },
  "frequency": "daily",
  "notify_email": true,
  "notify_in_app": true,
  "is_active": true
}
```

**Expected Results**:
- HTTP 201 Created
- Response contains alert ID
- Alert appears in database with `is_active=true`
- `created_at` and `updated_at` timestamps set

**Test Steps**:
```bash
# Get watchlist ID first
curl -X GET "http://localhost:8080/api/v1/watchlists" \
  -H "Authorization: Bearer $TOKEN"

# Create alert
curl -X POST "http://localhost:8080/api/v1/alerts" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "watch_list_id": "uuid-here",
    "symbol": "AAPL",
    "name": "AAPL Price Above $150",
    "alert_type": "price_above",
    "conditions": {"threshold": 150.00},
    "frequency": "daily",
    "notify_email": true,
    "notify_in_app": true
  }'
```

#### Test 1.2: Create Volume Above Alert
**Request Body**:
```json
{
  "watch_list_id": "{{watchlist_id}}",
  "symbol": "TSLA",
  "name": "TSLA High Volume Alert",
  "alert_type": "volume_above",
  "conditions": {
    "threshold": 50000000
  },
  "frequency": "always",
  "notify_email": false,
  "notify_in_app": true,
  "is_active": true
}
```

**Expected Results**: Same as Test 1.1

#### Test 1.3: Create Alert with Invalid Symbol
**Request Body**: Same as Test 1.1 but with `"symbol": "INVALID123"`

**Expected Results**:
- HTTP 400 Bad Request
- Error message: "symbol not found in watch list"

#### Test 1.4: Create Alert at Subscription Limit
**Setup**: Free account with 10 alerts already created

**Expected Results**:
- HTTP 403 Forbidden
- Error message: "Alert limit reached for your subscription"

#### Test 1.5: Create Alert with Invalid Frequency
**Request Body**: Same as Test 1.1 but with `"frequency": "invalid_frequency"`

**Expected Results**:
- HTTP 400 Bad Request
- Error message contains "invalid frequency"

---

### Section 2: Alert Retrieval (API Tests)

#### Test 2.1: List All Alerts
**Endpoint**: `GET /api/v1/alerts`

**Expected Results**:
- HTTP 200 OK
- Array of alert objects
- Each alert includes: id, name, symbol, alert_type, conditions, frequency, is_active, etc.

```bash
curl -X GET "http://localhost:8080/api/v1/alerts" \
  -H "Authorization: Bearer $TOKEN"
```

#### Test 2.2: List Alerts Filtered by Watch List
**Endpoint**: `GET /api/v1/alerts?watch_list_id={{watchlist_id}}`

**Expected Results**:
- HTTP 200 OK
- Only alerts for specified watchlist returned

#### Test 2.3: List Active Alerts Only
**Endpoint**: `GET /api/v1/alerts?is_active=true`

**Expected Results**:
- HTTP 200 OK
- Only alerts with `is_active=true` returned

#### Test 2.4: Get Single Alert
**Endpoint**: `GET /api/v1/alerts/{{alert_id}}`

**Expected Results**:
- HTTP 200 OK
- Alert object with all details
- `watch_list` object included with list details

#### Test 2.5: Get Alert Logs
**Endpoint**: `GET /api/v1/alerts/{{alert_id}}/logs`

**Expected Results**:
- HTTP 200 OK
- Array of alert trigger events
- Each log includes: triggered_at, condition_met_data, market_data_snapshot

---

### Section 3: Alert Updates (API Tests)

#### Test 3.1: Update Alert Name and Threshold
**Endpoint**: `PUT /api/v1/alerts/{{alert_id}}`

**Request Body**:
```json
{
  "name": "AAPL Price Above $160 (Updated)",
  "conditions": {
    "threshold": 160.00
  }
}
```

**Expected Results**:
- HTTP 200 OK
- Alert updated in database
- `updated_at` timestamp changed

#### Test 3.2: Toggle Alert Active Status
**Endpoint**: `PUT /api/v1/alerts/{{alert_id}}`

**Request Body**:
```json
{
  "is_active": false
}
```

**Expected Results**:
- HTTP 200 OK
- Alert `is_active` set to false
- Alert no longer triggers on next processing cycle

#### Test 3.3: Update Alert Frequency
**Endpoint**: `PUT /api/v1/alerts/{{alert_id}}`

**Request Body**:
```json
{
  "frequency": "once"
}
```

**Expected Results**:
- HTTP 200 OK
- Frequency updated
- Alert respects new frequency on next trigger

---

### Section 4: Alert Deletion (API Tests)

#### Test 4.1: Delete Alert
**Endpoint**: `DELETE /api/v1/alerts/{{alert_id}}`

**Expected Results**:
- HTTP 204 No Content
- Alert removed from database
- Associated logs preserved (optional, verify with team)

#### Test 4.2: Delete Non-existent Alert
**Endpoint**: `DELETE /api/v1/alerts/invalid-uuid`

**Expected Results**:
- HTTP 404 Not Found

---

### Section 5: Alert Triggering Logic (Backend Tests)

#### Test 5.1: Price Above Alert - Should Trigger
**Setup**:
- Alert: AAPL price above $100
- Current AAPL price: $150 (from Polygon API)

**Test Steps**:
```bash
# Manually trigger alert processor
cd backend
go run cmd/test-alert-processor/main.go --alert-id={{alert_id}}
```

**Expected Results**:
- Alert evaluates to `true`
- Alert log created in database
- Notification sent (if enabled)
- `last_triggered_at` timestamp updated
- `trigger_count` incremented

#### Test 5.2: Price Above Alert - Should NOT Trigger
**Setup**:
- Alert: AAPL price above $200
- Current AAPL price: $150

**Expected Results**:
- Alert evaluates to `false`
- No log created
- No notification sent
- Timestamps unchanged

#### Test 5.3: Volume Above Alert - Should Trigger
**Setup**:
- Alert: TSLA volume above 10M
- Current TSLA volume: 50M

**Expected Results**: Same as Test 5.1

#### Test 5.4: Frequency Enforcement - "once" Alert
**Setup**:
- Alert with frequency="once"
- Alert has `last_triggered_at` = yesterday

**Expected Results**:
- Alert does NOT trigger even if condition met
- Log message: "Skipping alert - frequency restriction"

#### Test 5.5: Frequency Enforcement - "daily" Alert (< 24 hours)
**Setup**:
- Alert with frequency="daily"
- Alert has `last_triggered_at` = 2 hours ago

**Expected Results**:
- Alert does NOT trigger even if condition met

#### Test 5.6: Frequency Enforcement - "daily" Alert (> 24 hours)
**Setup**:
- Alert with frequency="daily"
- Alert has `last_triggered_at` = 25 hours ago

**Expected Results**:
- Alert DOES trigger if condition met

#### Test 5.7: Frequency Enforcement - "always" Alert
**Setup**:
- Alert with frequency="always"
- Alert has `last_triggered_at` = 1 minute ago

**Expected Results**:
- Alert DOES trigger if condition met (no frequency restriction)

---

### Section 6: Batch Alert Processing (Performance Tests)

#### Test 6.1: Process 100 Alerts Efficiently
**Setup**:
- Create 100 active alerts across 50 unique symbols
- Mix of price and volume alerts

**Test Steps**:
```bash
cd backend
time go run cmd/alert-processor/main.go
```

**Expected Results**:
- All 100 alerts processed in < 2 seconds
- Only 2 Polygon API calls made (stocks + crypto batch)
- Console logs show:
  ```
  Found 100 active alerts to process
  Fetching quotes for 50 unique symbols...
  Received 50 quotes, processing alerts...
  Alert processing completed: 100 processed, X triggered
  ```

#### Test 6.2: Handle Missing Quote Data Gracefully
**Setup**:
- Create alert for symbol "TEST" (doesn't exist in Polygon)

**Expected Results**:
- Processing completes without error
- Log message: "Warning: No quote found for symbol TEST"
- Other alerts still process normally

---

### Section 7: Notification Delivery (Integration Tests)

#### Test 7.1: In-App Notification Created
**Setup**: Trigger an alert with `notify_in_app=true`

**Test Steps**:
```bash
# Check notifications endpoint
curl -X GET "http://localhost:8080/api/v1/notifications?type=alert_triggered" \
  -H "Authorization: Bearer $TOKEN"
```

**Expected Results**:
- HTTP 200 OK
- Notification object with:
  - `type: "alert_triggered"`
  - `title: "Alert: {alert_name}"`
  - `message: "Your alert for {symbol} has been triggered"`
  - `is_read: false`
  - `metadata` contains alert details

#### Test 7.2: Email Notification Sent
**Setup**: Trigger an alert with `notify_email=true`

**Expected Results**:
- Email sent to user's email address
- Email contains:
  - Alert name
  - Symbol and current price/volume
  - Condition that was met
  - Link to view alert details

**Manual Check**: Check email inbox or SendGrid dashboard

#### Test 7.3: Respect Quiet Hours
**Setup**:
- Set user quiet hours: 10 PM - 7 AM
- Trigger alert at 11 PM with `notify_email=true`

**Expected Results**:
- In-app notification still created
- Email NOT sent (quiet hours respected)

---

### Section 8: Subscription Limits (API Tests)

#### Test 8.1: Free Tier Limit Enforcement
**Setup**: Free account

**Test Steps**:
1. Get current subscription limits:
```bash
curl -X GET "http://localhost:8080/api/v1/subscriptions/limits" \
  -H "Authorization: Bearer $TOKEN"
```

**Expected Response**:
```json
{
  "max_watch_lists": 3,
  "current_watch_lists": 2,
  "max_watch_list_items": 10,
  "current_watch_list_items": 7,
  "max_alert_rules": 10,
  "current_alert_rules": 8
}
```

2. Try to create 11th alert

**Expected Results**: HTTP 403 Forbidden

#### Test 8.2: Premium Tier Limits
**Setup**: Premium account

**Expected Limits**:
- max_watch_lists: 20
- max_watch_list_items: 100
- max_alert_rules: 100

#### Test 8.3: Enterprise Tier Unlimited
**Setup**: Enterprise account

**Expected Limits**:
- All limits: -1 (unlimited)

---

### Section 9: Frontend UI Tests

#### Test 9.1: Alert List Page Load
**URL**: `http://localhost:3000/alerts`

**Test Steps**:
1. Navigate to alerts page
2. Verify authentication required
3. Login if needed
4. Observe page load

**Expected Results**:
- Page loads without errors
- Filter tabs visible: "All Alerts", "Active", "Inactive"
- Count badges show correct numbers
- Subscription limit display at top
- "Create Alert" button visible
- Alerts displayed in grid layout

#### Test 9.2: Filter Alerts by Status
**Test Steps**:
1. Click "Active" filter tab
2. Verify only active alerts shown
3. Click "Inactive" filter tab
4. Verify only inactive alerts shown

**Expected Results**:
- Filtering works correctly
- Count badges update
- URL parameter reflects filter (optional)

#### Test 9.3: Alert Card Display
**Expected Elements on Each Card**:
- ✅ Symbol (e.g., "AAPL")
- ✅ Alert name
- ✅ Alert type icon and label (e.g., "↑ Price Above")
- ✅ Condition details (e.g., "$150.00")
- ✅ Frequency badge (e.g., "Daily")
- ✅ Active/Inactive status badge
- ✅ Notification settings icons (email/bell)
- ✅ Last triggered time (if triggered)
- ✅ Trigger count
- ✅ Action buttons: Toggle, Edit, Delete

#### Test 9.4: Toggle Alert Active Status (Frontend)
**Test Steps**:
1. Click toggle switch on an alert card
2. Observe immediate UI update
3. Refresh page

**Expected Results**:
- Toggle switches state immediately
- Status badge updates (green → gray or vice versa)
- State persists after refresh

#### Test 9.5: Delete Alert with Confirmation
**Test Steps**:
1. Click delete button on alert card
2. Confirm deletion in modal
3. Observe alert removed

**Expected Results**:
- Confirmation modal appears
- After confirm, alert removed from list
- Success toast notification shown
- Count badges update

#### Test 9.6: Subscription Limit Warning
**Setup**: Free account with 8/10 alerts created

**Expected Results**:
- Warning banner shown: "You're using 8 of 10 alerts"
- Banner color: yellow/orange (80% usage)

#### Test 9.7: Subscription Limit Reached
**Setup**: Free account with 10/10 alerts created

**Test Steps**:
1. Click "Create Alert" button

**Expected Results**:
- Upgrade modal appears
- Modal explains limit reached
- "Upgrade to Premium" button visible
- Modal can be dismissed

#### Test 9.8: Empty State Display
**Setup**: Account with no alerts

**Expected Results**:
- Empty state illustration shown
- Message: "No alerts yet"
- "Create your first alert" button visible

---

### Section 10: Error Handling Tests

#### Test 10.1: Backend Unreachable
**Setup**: Stop backend server

**Expected Results**:
- Frontend shows error message
- Retry mechanism in place (optional)
- User-friendly error (not raw network error)

#### Test 10.2: Invalid JWT Token
**Setup**: Use expired or invalid token

**Expected Results**:
- HTTP 401 Unauthorized
- User redirected to login page

#### Test 10.3: Polygon API Down
**Setup**: Simulate Polygon API failure

**Expected Results**:
- Alert processing logs error but doesn't crash
- Other alerts continue processing
- Admin notification sent (optional)

---

### Section 11: Performance Tests

#### Test 11.1: Alert List Load Time
**Setup**: Account with 100 alerts

**Expected Results**:
- Page loads in < 2 seconds
- Pagination or virtual scrolling if needed

#### Test 11.2: Alert Processing Latency
**Setup**: 100 active alerts

**Measurements**:
- Total processing time: < 2 seconds
- API calls: Exactly 2 (stocks + crypto)
- Memory usage: < 100MB

#### Test 11.3: Concurrent User Load
**Setup**: 10 users creating alerts simultaneously

**Expected Results**:
- All requests succeed
- No database deadlocks
- Response time < 500ms per request

---

### Section 12: Security Tests

#### Test 12.1: Authorization Check
**Test Steps**:
1. User A creates alert
2. User B attempts to view User A's alert

**Expected Results**:
- HTTP 403 Forbidden
- Alert not accessible cross-user

#### Test 12.2: SQL Injection Attempt
**Test Steps**:
```bash
curl -X POST "http://localhost:8080/api/v1/alerts" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name": "Test\"; DROP TABLE alerts; --"}'
```

**Expected Results**:
- No SQL injection occurs
- Alert name stored as-is (escaped)

#### Test 12.3: XSS Prevention
**Test Steps**:
1. Create alert with name: `<script>alert('XSS')</script>`
2. View alert in frontend

**Expected Results**:
- Script not executed
- Name displayed as plain text

---

## Test Execution Tracking

### Test Progress Checklist

**Section 1: Alert Creation (API)**
- [ ] Test 1.1: Create Price Above Alert
- [ ] Test 1.2: Create Volume Above Alert
- [ ] Test 1.3: Invalid Symbol
- [ ] Test 1.4: Subscription Limit
- [ ] Test 1.5: Invalid Frequency

**Section 2: Alert Retrieval (API)**
- [ ] Test 2.1: List All Alerts
- [ ] Test 2.2: Filter by Watch List
- [ ] Test 2.3: Active Alerts Only
- [ ] Test 2.4: Get Single Alert
- [ ] Test 2.5: Get Alert Logs

**Section 3: Alert Updates (API)**
- [ ] Test 3.1: Update Name/Threshold
- [ ] Test 3.2: Toggle Active Status
- [ ] Test 3.3: Update Frequency

**Section 4: Alert Deletion (API)**
- [ ] Test 4.1: Delete Alert
- [ ] Test 4.2: Delete Non-existent

**Section 5: Alert Triggering Logic**
- [ ] Test 5.1: Price Above - Trigger
- [ ] Test 5.2: Price Above - No Trigger
- [ ] Test 5.3: Volume Above - Trigger
- [ ] Test 5.4: Frequency "once"
- [ ] Test 5.5: Frequency "daily" < 24h
- [ ] Test 5.6: Frequency "daily" > 24h
- [ ] Test 5.7: Frequency "always"

**Section 6: Batch Processing**
- [ ] Test 6.1: Process 100 Alerts
- [ ] Test 6.2: Handle Missing Quotes

**Section 7: Notifications**
- [ ] Test 7.1: In-App Notification
- [ ] Test 7.2: Email Notification
- [ ] Test 7.3: Quiet Hours

**Section 8: Subscription Limits**
- [ ] Test 8.1: Free Tier
- [ ] Test 8.2: Premium Tier
- [ ] Test 8.3: Enterprise Tier

**Section 9: Frontend UI**
- [ ] Test 9.1: Page Load
- [ ] Test 9.2: Filtering
- [ ] Test 9.3: Card Display
- [ ] Test 9.4: Toggle Active
- [ ] Test 9.5: Delete Alert
- [ ] Test 9.6: Limit Warning
- [ ] Test 9.7: Limit Reached
- [ ] Test 9.8: Empty State

**Section 10: Error Handling**
- [ ] Test 10.1: Backend Unreachable
- [ ] Test 10.2: Invalid Token
- [ ] Test 10.3: Polygon API Down

**Section 11: Performance**
- [ ] Test 11.1: Load Time
- [ ] Test 11.2: Processing Latency
- [ ] Test 11.3: Concurrent Load

**Section 12: Security**
- [ ] Test 12.1: Authorization
- [ ] Test 12.2: SQL Injection
- [ ] Test 12.3: XSS Prevention

---

## Bug Reporting Template

When a test fails, use this template to report bugs:

```markdown
**Bug ID**: BUG-ALERT-XXX
**Severity**: Critical / High / Medium / Low
**Test Case**: [Test number and name]
**Environment**: Local / Staging / Production

**Description**:
[Clear description of the issue]

**Steps to Reproduce**:
1. [Step 1]
2. [Step 2]
3. [Step 3]

**Expected Result**:
[What should happen]

**Actual Result**:
[What actually happened]

**Screenshots/Logs**:
[Attach relevant screenshots or log output]

**Reproduction Rate**: [Always / Sometimes / Rarely]
```

---

## Test Environment Cleanup

After testing, clean up test data:

```sql
-- Delete test alerts
DELETE FROM alert_rules WHERE user_id IN (
  SELECT id FROM users WHERE email LIKE 'test-%@investorcenter.ai'
);

-- Delete test notifications
DELETE FROM notifications WHERE user_id IN (
  SELECT id FROM users WHERE email LIKE 'test-%@investorcenter.ai'
);

-- Delete test logs
DELETE FROM alert_trigger_logs WHERE alert_rule_id IN (
  SELECT id FROM alert_rules WHERE user_id IN (
    SELECT id FROM users WHERE email LIKE 'test-%@investorcenter.ai'
  )
);
```

---

## Success Criteria

The alert system passes testing if:

1. ✅ All API endpoints return correct status codes and data
2. ✅ All alert types evaluate conditions correctly
3. ✅ Frequency enforcement works as designed
4. ✅ Notifications are delivered reliably
5. ✅ Subscription limits are enforced correctly
6. ✅ Frontend UI is functional and user-friendly
7. ✅ Performance meets requirements (< 2s for 100 alerts)
8. ✅ No critical or high severity bugs
9. ✅ Security tests pass without vulnerabilities
10. ✅ Error handling is graceful and informative

---

## Test Schedule

**Estimated Time**: 8-12 hours

**Day 1 (4 hours)**:
- Setup test environment
- Execute API tests (Sections 1-4)
- Execute triggering logic tests (Section 5)

**Day 2 (4 hours)**:
- Execute batch processing tests (Section 6)
- Execute notification tests (Section 7)
- Execute subscription limit tests (Section 8)

**Day 3 (4 hours)**:
- Execute frontend UI tests (Section 9)
- Execute error handling tests (Section 10)
- Execute performance tests (Section 11)
- Execute security tests (Section 12)
- Bug fixing and retesting

---

## Notes

- **Automation**: Many of these tests can be automated using Postman, Jest, or Go test frameworks
- **CI/CD Integration**: Consider running automated tests on every deployment
- **Monitoring**: Set up production monitoring for alert processing latency and success rates
- **User Acceptance**: Have product team do final UAT before production release

---

*Generated: 2025-11-04*
*Version: 1.0*
*Author: Claude (Anthropic)*
