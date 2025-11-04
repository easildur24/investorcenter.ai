package database

import (
	"database/sql"
	"errors"
	"fmt"
	"investorcenter-api/models"
)

// Subscription Plan Operations

// GetAllSubscriptionPlans retrieves all active subscription plans
func GetAllSubscriptionPlans() ([]models.SubscriptionPlan, error) {
	query := `
		SELECT
			id, name, display_name, description, price_monthly, price_yearly,
			max_watch_lists, max_items_per_watch_list, max_alert_rules,
			max_heatmap_configs, features, is_active, created_at, updated_at
		FROM subscription_plans
		WHERE is_active = true
		ORDER BY price_monthly ASC
	`
	rows, err := DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription plans: %w", err)
	}
	defer rows.Close()

	plans := []models.SubscriptionPlan{}
	for rows.Next() {
		var plan models.SubscriptionPlan
		err := rows.Scan(
			&plan.ID,
			&plan.Name,
			&plan.DisplayName,
			&plan.Description,
			&plan.PriceMonthly,
			&plan.PriceYearly,
			&plan.MaxWatchLists,
			&plan.MaxItemsPerWatchList,
			&plan.MaxAlertRules,
			&plan.MaxHeatmapConfigs,
			&plan.Features,
			&plan.IsActive,
			&plan.CreatedAt,
			&plan.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan subscription plan: %w", err)
		}
		plans = append(plans, plan)
	}

	return plans, nil
}

// GetSubscriptionPlanByID retrieves a specific subscription plan
func GetSubscriptionPlanByID(planID string) (*models.SubscriptionPlan, error) {
	query := `
		SELECT
			id, name, display_name, description, price_monthly, price_yearly,
			max_watch_lists, max_items_per_watch_list, max_alert_rules,
			max_heatmap_configs, features, is_active, created_at, updated_at
		FROM subscription_plans
		WHERE id = $1
	`
	plan := &models.SubscriptionPlan{}
	err := DB.QueryRow(query, planID).Scan(
		&plan.ID,
		&plan.Name,
		&plan.DisplayName,
		&plan.Description,
		&plan.PriceMonthly,
		&plan.PriceYearly,
		&plan.MaxWatchLists,
		&plan.MaxItemsPerWatchList,
		&plan.MaxAlertRules,
		&plan.MaxHeatmapConfigs,
		&plan.Features,
		&plan.IsActive,
		&plan.CreatedAt,
		&plan.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("subscription plan not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription plan: %w", err)
	}

	return plan, nil
}

// GetSubscriptionPlanByName retrieves a subscription plan by name
func GetSubscriptionPlanByName(name string) (*models.SubscriptionPlan, error) {
	query := `
		SELECT
			id, name, display_name, description, price_monthly, price_yearly,
			max_watch_lists, max_items_per_watch_list, max_alert_rules,
			max_heatmap_configs, features, is_active, created_at, updated_at
		FROM subscription_plans
		WHERE name = $1
	`
	plan := &models.SubscriptionPlan{}
	err := DB.QueryRow(query, name).Scan(
		&plan.ID,
		&plan.Name,
		&plan.DisplayName,
		&plan.Description,
		&plan.PriceMonthly,
		&plan.PriceYearly,
		&plan.MaxWatchLists,
		&plan.MaxItemsPerWatchList,
		&plan.MaxAlertRules,
		&plan.MaxHeatmapConfigs,
		&plan.Features,
		&plan.IsActive,
		&plan.CreatedAt,
		&plan.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("subscription plan not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription plan: %w", err)
	}

	return plan, nil
}

// User Subscription Operations

// CreateUserSubscription creates a new subscription for a user
func CreateUserSubscription(subscription *models.UserSubscription) error {
	query := `
		INSERT INTO user_subscriptions (
			user_id, plan_id, status, billing_period, current_period_start,
			current_period_end, stripe_subscription_id, stripe_customer_id,
			payment_method, next_payment_date
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, started_at, created_at, updated_at
	`
	err := DB.QueryRow(
		query,
		subscription.UserID,
		subscription.PlanID,
		subscription.Status,
		subscription.BillingPeriod,
		subscription.CurrentPeriodStart,
		subscription.CurrentPeriodEnd,
		subscription.StripeSubscriptionID,
		subscription.StripeCustomerID,
		subscription.PaymentMethod,
		subscription.NextPaymentDate,
	).Scan(&subscription.ID, &subscription.StartedAt, &subscription.CreatedAt, &subscription.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create user subscription: %w", err)
	}
	return nil
}

// GetUserSubscription retrieves active subscription for a user
func GetUserSubscription(userID string) (*models.UserSubscriptionWithPlan, error) {
	query := `
		SELECT
			us.id, us.user_id, us.plan_id, us.status, us.billing_period,
			us.started_at, us.current_period_start, us.current_period_end,
			us.canceled_at, us.ended_at, us.stripe_subscription_id,
			us.stripe_customer_id, us.payment_method, us.last_payment_date,
			us.next_payment_date, us.created_at, us.updated_at,
			sp.name as plan_name, sp.display_name as plan_display_name,
			sp.features as plan_features, sp.max_watch_lists,
			sp.max_items_per_watch_list, sp.max_alert_rules, sp.max_heatmap_configs
		FROM user_subscriptions us
		JOIN subscription_plans sp ON us.plan_id = sp.id
		WHERE us.user_id = $1
		ORDER BY us.created_at DESC
		LIMIT 1
	`
	sub := &models.UserSubscriptionWithPlan{}
	err := DB.QueryRow(query, userID).Scan(
		&sub.ID,
		&sub.UserID,
		&sub.PlanID,
		&sub.Status,
		&sub.BillingPeriod,
		&sub.StartedAt,
		&sub.CurrentPeriodStart,
		&sub.CurrentPeriodEnd,
		&sub.CanceledAt,
		&sub.EndedAt,
		&sub.StripeSubscriptionID,
		&sub.StripeCustomerID,
		&sub.PaymentMethod,
		&sub.LastPaymentDate,
		&sub.NextPaymentDate,
		&sub.CreatedAt,
		&sub.UpdatedAt,
		&sub.PlanName,
		&sub.PlanDisplayName,
		&sub.PlanFeatures,
		&sub.MaxWatchLists,
		&sub.MaxItemsPerWatchList,
		&sub.MaxAlertRules,
		&sub.MaxHeatmapConfigs,
	)

	if err == sql.ErrNoRows {
		// User has no subscription, return free tier defaults
		freePlan, err := GetSubscriptionPlanByName("free")
		if err != nil {
			return nil, err
		}
		return &models.UserSubscriptionWithPlan{
			UserSubscription: models.UserSubscription{
				UserID: userID,
				Status: "active",
			},
			PlanName:             freePlan.Name,
			PlanDisplayName:      freePlan.DisplayName,
			PlanFeatures:         freePlan.Features,
			MaxWatchLists:        freePlan.MaxWatchLists,
			MaxItemsPerWatchList: freePlan.MaxItemsPerWatchList,
			MaxAlertRules:        freePlan.MaxAlertRules,
			MaxHeatmapConfigs:    freePlan.MaxHeatmapConfigs,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user subscription: %w", err)
	}

	return sub, nil
}

// UpdateUserSubscription updates a user subscription
func UpdateUserSubscription(subscriptionID string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return errors.New("no fields to update")
	}

	query := "UPDATE user_subscriptions SET "
	args := []interface{}{}
	argCount := 0

	for field, value := range updates {
		argCount++
		if argCount > 1 {
			query += ", "
		}
		query += fmt.Sprintf("%s = $%d", field, argCount)
		args = append(args, value)
	}

	argCount++
	query += fmt.Sprintf(" WHERE id = $%d", argCount)
	args = append(args, subscriptionID)

	result, err := DB.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update user subscription: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("user subscription not found")
	}

	return nil
}

// CancelUserSubscription cancels a user subscription
func CancelUserSubscription(userID string) error {
	query := `
		UPDATE user_subscriptions
		SET status = 'canceled', canceled_at = CURRENT_TIMESTAMP
		WHERE user_id = $1 AND status = 'active'
	`
	result, err := DB.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to cancel user subscription: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("active subscription not found")
	}

	return nil
}

// GetUserSubscriptionLimits gets the limits for a user
func GetUserSubscriptionLimits(userID string) (*models.SubscriptionLimits, error) {
	sub, err := GetUserSubscription(userID)
	if err != nil {
		return nil, err
	}

	return &models.SubscriptionLimits{
		MaxWatchLists:        sub.MaxWatchLists,
		MaxItemsPerWatchList: sub.MaxItemsPerWatchList,
		MaxAlertRules:        sub.MaxAlertRules,
		MaxHeatmapConfigs:    sub.MaxHeatmapConfigs,
		Features:             sub.PlanFeatures,
	}, nil
}

// Payment History Operations

// CreatePaymentHistory records a payment transaction
func CreatePaymentHistory(payment *models.PaymentHistory) error {
	query := `
		INSERT INTO payment_history (
			user_id, subscription_id, amount, currency, status,
			payment_method, stripe_payment_intent_id, stripe_invoice_id,
			description, receipt_url
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at
	`
	err := DB.QueryRow(
		query,
		payment.UserID,
		payment.SubscriptionID,
		payment.Amount,
		payment.Currency,
		payment.Status,
		payment.PaymentMethod,
		payment.StripePaymentIntentID,
		payment.StripeInvoiceID,
		payment.Description,
		payment.ReceiptURL,
	).Scan(&payment.ID, &payment.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create payment history: %w", err)
	}
	return nil
}

// GetPaymentHistory retrieves payment history for a user
func GetPaymentHistory(userID string, limit int) ([]models.PaymentHistory, error) {
	query := `
		SELECT
			id, user_id, subscription_id, amount, currency, status,
			payment_method, stripe_payment_intent_id, stripe_invoice_id,
			description, receipt_url, created_at
		FROM payment_history
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	args := []interface{}{userID}

	if limit > 0 {
		query += " LIMIT $2"
		args = append(args, limit)
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment history: %w", err)
	}
	defer rows.Close()

	payments := []models.PaymentHistory{}
	for rows.Next() {
		var payment models.PaymentHistory
		err := rows.Scan(
			&payment.ID,
			&payment.UserID,
			&payment.SubscriptionID,
			&payment.Amount,
			&payment.Currency,
			&payment.Status,
			&payment.PaymentMethod,
			&payment.StripePaymentIntentID,
			&payment.StripeInvoiceID,
			&payment.Description,
			&payment.ReceiptURL,
			&payment.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan payment history: %w", err)
		}
		payments = append(payments, payment)
	}

	return payments, nil
}
