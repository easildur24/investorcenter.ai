package delivery

import (
	"errors"
	"fmt"
	"log"

	"notification-service/models"
)

// Router dispatches notifications to the appropriate delivery channels
// based on the alert rule's configuration.
type Router struct {
	inApp *InAppDelivery
	email *EmailDelivery
}

// NewRouter creates a new delivery Router.
func NewRouter(email *EmailDelivery, inApp *InAppDelivery) *Router {
	return &Router{inApp: inApp, email: email}
}

// Deliver sends notifications for a triggered alert via all configured channels.
// Errors from individual channels are collected but do not block other channels.
func (r *Router) Deliver(alert *models.AlertRule, alertLog *models.AlertLog, quote *models.SymbolQuote) error {
	var errs []error

	// In-app notification
	if alert.NotifyInApp {
		if err := r.inApp.Send(alert, alertLog, quote); err != nil {
			errs = append(errs, fmt.Errorf("in-app: %w", err))
			log.Printf("In-app delivery failed for alert %s: %v", alert.ID, err)
		}
	}

	// Email notification
	if alert.NotifyEmail {
		if err := r.email.Send(alert, alertLog, quote); err != nil {
			errs = append(errs, fmt.Errorf("email: %w", err))
			log.Printf("Email delivery failed for alert %s: %v", alert.ID, err)
		}
	}

	return errors.Join(errs...)
}
