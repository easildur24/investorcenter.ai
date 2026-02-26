package delivery

import (
	"fmt"
	"log"

	"notification-service/models"
)

// Router dispatches notifications to the appropriate delivery channels
// based on the alert rule's configuration.
type Router struct {
	email *EmailDelivery
}

// NewRouter creates a new delivery Router.
func NewRouter(email *EmailDelivery) *Router {
	return &Router{email: email}
}

// Deliver sends notifications for a triggered alert via configured channels.
func (r *Router) Deliver(alert *models.AlertRule, alertLog *models.AlertLog, quote *models.SymbolQuote) error {
	// Email notification
	if alert.NotifyEmail {
		if err := r.email.Send(alert, alertLog, quote); err != nil {
			log.Printf("Email delivery failed for alert %s: %v", alert.ID, err)
			return fmt.Errorf("email: %w", err)
		}
	}

	return nil
}
