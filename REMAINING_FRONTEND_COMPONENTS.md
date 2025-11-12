# Remaining Frontend Components

## ✅ COMPLETED
1. **Alert List Page** - `app/alerts/page.tsx` ✅
   - Filter tabs (all/active/inactive)
   - Subscription limit display
   - Empty states
   - Loading and error states
   - Create alert button with limit checking

2. **Alert Card Component** - `components/alerts/AlertCard.tsx` ✅
   - Full alert details display
   - Toggle active/inactive
   - Edit and delete actions
   - Notification settings display
   - Trigger statistics

3. **API Clients** - `lib/api/*.ts` ✅
   - alerts.ts - Alert CRUD operations
   - notifications.ts - Notification management
   - subscriptions.ts - Subscription and billing

## ⏳ REMAINING TO CREATE

### High Priority (Core Functionality)

#### 1. Create Alert Modal
**File**: `components/alerts/CreateAlertModal.tsx`
**Features**:
- Multi-step wizard (Select watch list → Select symbol → Configure alert)
- Alert type selection with visual icons
- Dynamic condition inputs based on alert type
- Frequency selection
- Notification preferences
- Form validation
- Error handling

**Template Structure**:
```tsx
export default function CreateAlertModal({ onClose, onSuccess }) {
  const [step, setStep] = useState(1); // 1: Watch list, 2: Symbol, 3: Config
  const [selectedWatchList, setSelectedWatchList] = useState(null);
  const [selectedSymbol, setSelectedSymbol] = useState(null);
  const [alertType, setAlertType] = useState('');
  const [conditions, setConditions] = useState({});
  // ... form state management

  return (
    <Modal>
      {step === 1 && <SelectWatchList />}
      {step === 2 && <SelectSymbol />}
      {step === 3 && <ConfigureAlert />}
    </Modal>
  );
}
```

#### 2. Edit Alert Modal
**File**: `components/alerts/EditAlertModal.tsx`
**Features**:
- Pre-populated form with current alert settings
- Update name, description
- Modify conditions
- Change frequency
- Update notification preferences
- Validation

#### 3. Notification Center
**File**: `components/NotificationCenter.tsx`
**Features**:
- Bell icon with unread count badge in header
- Dropdown panel with notifications list
- Real-time updates (polling or WebSocket)
- Mark as read/dismiss
- Click to view alert details
- Empty state
- Load more pagination

**Usage**: Add to main layout/header

#### 4. Notification Preferences Page
**File**: `app/settings/notifications/page.tsx`
**Features**:
- Email notifications toggle
- In-app notifications toggle
- Quiet hours configuration
- Alert type preferences (price/volume/news)
- Digest frequency selection
- Save button

#### 5. Subscription/Pricing Page
**File**: `app/pricing/page.tsx`
**Features**:
- Three-tier pricing cards (Free/Premium/Enterprise)
- Feature comparison table
- Monthly/Yearly toggle
- Current plan indicator
- Upgrade/Downgrade buttons
- Billing history table (for subscribed users)

**Layout**:
```
┌─────────────────┬─────────────────┬─────────────────┐
│   Free Plan     │  Premium Plan   │ Enterprise Plan │
│   $0/month      │  $19.99/month   │  $99.99/month   │
│                 │                 │                 │
│ ✓ 3 lists       │ ✓ 20 lists      │ ✓ Unlimited     │
│ ✓ 10 items      │ ✓ 100 items     │ ✓ Unlimited     │
│ ✓ 10 alerts     │ ✓ 100 alerts    │ ✓ Unlimited     │
│                 │ ✓ Real-time     │ ✓ API Access    │
│                 │ ✓ Advanced      │ ✓ White Label   │
└─────────────────┴─────────────────┴─────────────────┘
```

#### 6. Upgrade Modal
**File**: `components/subscription/UpgradeModal.tsx`
**Features**:
- Triggered when limit reached
- Shows current plan limits
- Displays upgrade benefits
- CTA button to pricing page
- Dismissible

### Medium Priority (Enhanced UX)

#### 7. Alert Logs Page
**File**: `app/alerts/logs/page.tsx`
**Features**:
- List of all triggered alerts
- Filter by symbol, alert type, date range
- Mark as read
- Dismiss
- View market data snapshot
- Pagination

#### 8. Toast Notification Component
**File**: `components/Toast.tsx`
**Features**:
- Success/error/info/warning types
- Auto-dismiss after timeout
- Manual dismiss button
- Stacking multiple toasts
- Animation

**Already exists** - Check if it needs updating

#### 9. Condition Builder Component
**File**: `components/alerts/ConditionBuilder.tsx`
**Features**:
- Dynamic form based on alert type
- Price threshold inputs
- Percentage change inputs
- Volume multiplier
- Direction selectors
- Validation

### Low Priority (Nice to Have)

#### 10. Alert Template Library
**File**: `components/alerts/AlertTemplates.tsx`
**Features**:
- Pre-configured alert templates
- "Price drops 10%" template
- "Volume spike 2x" template
- One-click apply

#### 11. Alert Performance Dashboard
**File**: `app/alerts/analytics/page.tsx`
**Features**:
- Trigger frequency charts
- Most triggered alerts
- Performance metrics
- Success rate

#### 12. Bulk Alert Actions
**Features**:
- Select multiple alerts
- Bulk activate/deactivate
- Bulk delete
- Export to CSV

---

## Component Dependencies

### Shared Components Needed

1. **Modal Wrapper** (`components/Modal.tsx`)
```tsx
export default function Modal({ children, onClose, title, size = 'md' }) {
  return (
    <div className="fixed inset-0 z-50 overflow-y-auto">
      <div className="flex items-center justify-center min-h-screen px-4">
        <div className="fixed inset-0 bg-black/50" onClick={onClose} />
        <div className={`relative bg-slate-800 rounded-lg shadow-xl ${sizeClasses[size]}`}>
          {/* Modal content */}
        </div>
      </div>
    </div>
  );
}
```

2. **Select Component** (`components/ui/Select.tsx`)
3. **Input Component** (`components/ui/Input.tsx`)
4. **Button Component** (`components/ui/Button.tsx`)
5. **Badge Component** (`components/ui/Badge.tsx`)

---

## Implementation Priority Order

1. ✅ Alert list page (DONE)
2. ✅ Alert card component (DONE)
3. **Create Alert Modal** - Critical for functionality
4. **Edit Alert Modal** - Critical for functionality
5. **Upgrade Modal** - Blocks users at limits
6. **Notification Center** - Core feature
7. **Pricing Page** - Revenue generation
8. **Notification Preferences** - User control
9. Alert Logs Page - History tracking
10. Toast notifications - UX enhancement

---

## Quick Start Implementation Guide

### Step 1: Create Modal Wrapper
```bash
touch components/Modal.tsx
# Implement basic modal with backdrop and close functionality
```

### Step 2: Create Alert Modals
```bash
touch components/alerts/CreateAlertModal.tsx
touch components/alerts/EditAlertModal.tsx
# Implement form logic with alertAPI
```

### Step 3: Create Subscription Components
```bash
touch components/subscription/UpgradeModal.tsx
touch app/pricing/page.tsx
# Implement pricing display and upgrade flow
```

### Step 4: Create Notification Center
```bash
touch components/NotificationCenter.tsx
# Add to layout, implement polling for updates
```

### Step 5: Integration
- Add notification center to main layout header
- Add navigation links to alerts page
- Test full flow: Create → Trigger → Notify → View

---

## Styling Guidelines

All components should follow the existing design system:
- **Background**: `bg-slate-800/50` with `backdrop-blur-sm`
- **Borders**: `border-slate-700` or `border-purple-500/50` for active
- **Text**: `text-white` for primary, `text-gray-300/400` for secondary
- **Buttons**: Gradient from `purple-600` to `blue-600`
- **Cards**: Rounded with `rounded-lg`, padding `p-6`
- **Icons**: Use Heroicons (already in use)

---

## Testing Checklist

### Alert Creation Flow:
- [ ] Open create modal
- [ ] Select watch list (shows items)
- [ ] Select symbol from list
- [ ] Choose alert type
- [ ] Configure conditions
- [ ] Set frequency
- [ ] Enable/disable notifications
- [ ] Submit and verify alert created
- [ ] Check subscription limit enforcement

### Alert Management:
- [ ] View alerts list
- [ ] Filter by active/inactive
- [ ] Edit alert
- [ ] Toggle active status
- [ ] Delete alert with confirmation
- [ ] View trigger history

### Notifications:
- [ ] Receive in-app notification when alert triggers
- [ ] Mark notification as read
- [ ] Dismiss notification
- [ ] View unread count badge
- [ ] Configure notification preferences

### Subscription:
- [ ] View current plan limits
- [ ] Hit alert limit → See upgrade modal
- [ ] View pricing page
- [ ] See feature comparison
- [ ] (Future) Complete upgrade flow with Stripe

---

## Estimated Implementation Time

| Component | Complexity | Time Estimate |
|-----------|------------|---------------|
| CreateAlertModal | High | 4-6 hours |
| EditAlertModal | Medium | 2-3 hours |
| UpgradeModal | Low | 1-2 hours |
| NotificationCenter | Medium | 3-4 hours |
| PricingPage | Medium | 2-3 hours |
| NotificationPreferences | Low | 2 hours |
| AlertLogsPage | Medium | 2-3 hours |
| **TOTAL** | - | **16-23 hours** |

---

## Notes

- All modals should be dismissible with ESC key
- Forms should have loading states during submission
- Success/error messages should be shown via toast notifications
- All API calls should have proper error handling
- Components should be responsive (mobile-friendly)
- Use React hooks for state management
- Consider using React Hook Form for complex forms
- Add proper TypeScript types for all props

---

*Generated: 2025-11-04*
*Status: Alert list and card components completed, modals and other components pending*
