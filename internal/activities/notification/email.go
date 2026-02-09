package notification

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Youmanvi/taskorchestrator/internal/pkg/errors"
)

// EmailNotificationInput is the input for sending email notifications
type EmailNotificationInput struct {
	CustomerEmail string
	OrderID       string
	EventType     string // "order_confirmed", "order_failed", "refund_issued"
}

// EmailNotificationOutput is the output of sending email
type EmailNotificationOutput struct {
	MessageID string
	Status    string
}

// EmailService sends email notifications
type EmailService interface {
	SendEmail(ctx context.Context, to, subject, body string) (string, error)
}

// SendOrderConfirmationActivity sends order confirmation email
func SendOrderConfirmationActivity(emailService EmailService) func(ctx context.Context, input []byte) ([]byte, error) {
	return func(ctx context.Context, input []byte) ([]byte, error) {
		var inp EmailNotificationInput
		if err := json.Unmarshal(input, &inp); err != nil {
			return nil, errors.NewPermanentError("INVALID_INPUT", "failed to unmarshal email input", err)
		}

		if inp.CustomerEmail == "" {
			return nil, errors.NewPermanentError("MISSING_EMAIL", "customer email is required", nil)
		}

		messageID, err := emailService.SendEmail(
			ctx,
			inp.CustomerEmail,
			fmt.Sprintf("Order %s Confirmed", inp.OrderID),
			fmt.Sprintf("Your order %s has been confirmed and is being processed.", inp.OrderID),
		)
		if err != nil {
			return nil, errors.NewTransientError("EMAIL_SEND_FAILED", "failed to send confirmation email", err)
		}

		output := EmailNotificationOutput{
			MessageID: messageID,
			Status:    "sent",
		}

		result, err := json.Marshal(output)
		if err != nil {
			return nil, errors.NewPermanentError("SERIALIZATION_ERROR", "failed to marshal email output", err)
		}

		return result, nil
	}
}

// SendOrderFailureActivity sends order failure notification
func SendOrderFailureActivity(emailService EmailService) func(ctx context.Context, input []byte) ([]byte, error) {
	return func(ctx context.Context, input []byte) ([]byte, error) {
		var inp EmailNotificationInput
		if err := json.Unmarshal(input, &inp); err != nil {
			return nil, errors.NewPermanentError("INVALID_INPUT", "failed to unmarshal email input", err)
		}

		if inp.CustomerEmail == "" {
			return nil, errors.NewPermanentError("MISSING_EMAIL", "customer email is required", nil)
		}

		messageID, err := emailService.SendEmail(
			ctx,
			inp.CustomerEmail,
			fmt.Sprintf("Order %s Failed", inp.OrderID),
			fmt.Sprintf("Unfortunately, your order %s could not be processed. Please try again.", inp.OrderID),
		)
		if err != nil {
			return nil, errors.NewTransientError("EMAIL_SEND_FAILED", "failed to send failure email", err)
		}

		output := EmailNotificationOutput{
			MessageID: messageID,
			Status:    "sent",
		}

		result, err := json.Marshal(output)
		if err != nil {
			return nil, errors.NewPermanentError("SERIALIZATION_ERROR", "failed to marshal email output", err)
		}

		return result, nil
	}
}

// SendRefundNotificationActivity sends refund notification
func SendRefundNotificationActivity(emailService EmailService) func(ctx context.Context, input []byte) ([]byte, error) {
	return func(ctx context.Context, input []byte) ([]byte, error) {
		var inp EmailNotificationInput
		if err := json.Unmarshal(input, &inp); err != nil {
			return nil, errors.NewPermanentError("INVALID_INPUT", "failed to unmarshal email input", err)
		}

		if inp.CustomerEmail == "" {
			return nil, errors.NewPermanentError("MISSING_EMAIL", "customer email is required", nil)
		}

		messageID, err := emailService.SendEmail(
			ctx,
			inp.CustomerEmail,
			fmt.Sprintf("Refund Issued for Order %s", inp.OrderID),
			fmt.Sprintf("Your refund for order %s has been processed.", inp.OrderID),
		)
		if err != nil {
			return nil, errors.NewTransientError("EMAIL_SEND_FAILED", "failed to send refund email", err)
		}

		output := EmailNotificationOutput{
			MessageID: messageID,
			Status:    "sent",
		}

		result, err := json.Marshal(output)
		if err != nil {
			return nil, errors.NewPermanentError("SERIALIZATION_ERROR", "failed to marshal email output", err)
		}

		return result, nil
	}
}
