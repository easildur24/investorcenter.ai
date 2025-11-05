package models

import (
	"encoding/json"
	"time"
)

// SubscriptionPlan defines available subscription tiers
type SubscriptionPlan struct {
	ID                   string          `json:"id" db:"id"`
	Name                 string          `json:"name" db:"name"`
	DisplayName          string          `json:"display_name" db:"display_name"`
	Description          *string         `json:"description,omitempty" db:"description"`
	PriceMonthly         float64         `json:"price_monthly" db:"price_monthly"`
	PriceYearly          float64         `json:"price_yearly" db:"price_yearly"`
	MaxWatchLists        int             `json:"max_watch_lists" db:"max_watch_lists"`
	MaxItemsPerWatchList int             `json:"max_items_per_watch_list" db:"max_items_per_watch_list"`
	MaxAlertRules        int             `json:"max_alert_rules" db:"max_alert_rules"`
	MaxHeatmapConfigs    int             `json:"max_heatmap_configs" db:"max_heatmap_configs"`
	Features             json.RawMessage `json:"features" db:"features"`
	IsActive             bool            `json:"is_active" db:"is_active"`
	CreatedAt            time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at" db:"updated_at"`
}

// UserSubscription represents a user's active subscription
type UserSubscription struct {
	ID                   string     `json:"id" db:"id"`
	UserID               string     `json:"user_id" db:"user_id"`
	PlanID               string     `json:"plan_id" db:"plan_id"`
	Status               string     `json:"status" db:"status"`
	BillingPeriod        string     `json:"billing_period" db:"billing_period"`
	StartedAt            time.Time  `json:"started_at" db:"started_at"`
	CurrentPeriodStart   time.Time  `json:"current_period_start" db:"current_period_start"`
	CurrentPeriodEnd     *time.Time `json:"current_period_end,omitempty" db:"current_period_end"`
	CanceledAt           *time.Time `json:"canceled_at,omitempty" db:"canceled_at"`
	EndedAt              *time.Time `json:"ended_at,omitempty" db:"ended_at"`
	StripeSubscriptionID *string    `json:"stripe_subscription_id,omitempty" db:"stripe_subscription_id"`
	StripeCustomerID     *string    `json:"stripe_customer_id,omitempty" db:"stripe_customer_id"`
	PaymentMethod        *string    `json:"payment_method,omitempty" db:"payment_method"`
	LastPaymentDate      *time.Time `json:"last_payment_date,omitempty" db:"last_payment_date"`
	NextPaymentDate      *time.Time `json:"next_payment_date,omitempty" db:"next_payment_date"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at" db:"updated_at"`
}

// PaymentHistory tracks payment transactions
type PaymentHistory struct {
	ID                    string    `json:"id" db:"id"`
	UserID                string    `json:"user_id" db:"user_id"`
	SubscriptionID        *string   `json:"subscription_id,omitempty" db:"subscription_id"`
	Amount                float64   `json:"amount" db:"amount"`
	Currency              string    `json:"currency" db:"currency"`
	Status                string    `json:"status" db:"status"`
	PaymentMethod         *string   `json:"payment_method,omitempty" db:"payment_method"`
	StripePaymentIntentID *string   `json:"stripe_payment_intent_id,omitempty" db:"stripe_payment_intent_id"`
	StripeInvoiceID       *string   `json:"stripe_invoice_id,omitempty" db:"stripe_invoice_id"`
	Description           *string   `json:"description,omitempty" db:"description"`
	ReceiptURL            *string   `json:"receipt_url,omitempty" db:"receipt_url"`
	CreatedAt             time.Time `json:"created_at" db:"created_at"`
}

// UserSubscriptionWithPlan includes plan details
type UserSubscriptionWithPlan struct {
	UserSubscription
	PlanName             string          `json:"plan_name"`
	PlanDisplayName      string          `json:"plan_display_name"`
	PlanFeatures         json.RawMessage `json:"plan_features"`
	MaxWatchLists        int             `json:"max_watch_lists"`
	MaxItemsPerWatchList int             `json:"max_items_per_watch_list"`
	MaxAlertRules        int             `json:"max_alert_rules"`
	MaxHeatmapConfigs    int             `json:"max_heatmap_configs"`
}

// CreateSubscriptionRequest for initiating a subscription
type CreateSubscriptionRequest struct {
	PlanID        string `json:"plan_id" binding:"required"`
	BillingPeriod string `json:"billing_period" binding:"required"`
	PaymentMethod string `json:"payment_method"`
}

// UpdateSubscriptionRequest for modifying subscription
type UpdateSubscriptionRequest struct {
	PlanID        *string `json:"plan_id,omitempty"`
	BillingPeriod *string `json:"billing_period,omitempty"`
	PaymentMethod *string `json:"payment_method,omitempty"`
}

// SubscriptionLimits represents the limits for a user's plan
type SubscriptionLimits struct {
	MaxWatchLists        int             `json:"max_watch_lists"`
	MaxItemsPerWatchList int             `json:"max_items_per_watch_list"`
	MaxAlertRules        int             `json:"max_alert_rules"`
	MaxHeatmapConfigs    int             `json:"max_heatmap_configs"`
	Features             json.RawMessage `json:"features"`
}
