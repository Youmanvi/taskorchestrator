package workflows

import (
	"encoding/json"
	"fmt"

	"github.com/microsoft/durabletask-go/task"
	"github.com/Youmanvi/taskorchestrator/internal/activities/inventory"
	"github.com/Youmanvi/taskorchestrator/internal/activities/notification"
	"github.com/Youmanvi/taskorchestrator/internal/activities/payment"
	"github.com/Youmanvi/taskorchestrator/internal/domain"
)

// OrderProcessingInput is the input to the order processing orchestrator
type OrderProcessingInput struct {
	Order      domain.Order
	CustomerEmail string
}

// OrderProcessingOutput is the output of the order processing orchestrator
type OrderProcessingOutput struct {
	Status        string
	OrderID       string
	PaymentID     string
	ReservationID string
	Message       string
}

// OrderProcessingOrchestrator orchestrates the order processing workflow
func OrderProcessingOrchestrator(ctx *task.OrchestrationContext) (any, error) {
	var inp OrderProcessingInput
	if err := ctx.GetInput(&inp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal order processing input: %w", err)
	}

	order := inp.Order
	output := OrderProcessingOutput{
		OrderID: order.ID,
		Status:  "pending",
	}

	// Step 1: Check inventory availability
	checkInput := inventory.CheckAvailabilityInput{
		Items: order.Items,
	}
	checkInputBytes, _ := json.Marshal(checkInput)

	checkResult := ctx.CallActivity("inventory:check", task.WithActivityInput(checkInputBytes))
	var checkOutput inventory.CheckAvailabilityOutput
	if err := checkResult.Await(&checkOutput); err != nil {
		output.Status = "failed"
		output.Message = fmt.Sprintf("inventory check failed: %v", err)
		return output, nil
	}

	if !checkOutput.Available {
		output.Status = "failed"
		output.Message = "items not available"
		return output, nil
	}

	// Step 2: Reserve inventory
	reserveInput := inventory.ReserveInventoryInput{
		OrderID: order.ID,
		Items:   order.Items,
	}
	reserveInputBytes, _ := json.Marshal(reserveInput)

	reserveResult := ctx.CallActivity("inventory:reserve", task.WithActivityInput(reserveInputBytes))
	var reserveOutput inventory.ReserveInventoryOutput
	if err := reserveResult.Await(&reserveOutput); err != nil {
		output.Status = "failed"
		output.Message = fmt.Sprintf("inventory reservation failed: %v", err)
		return output, nil
	}

	output.ReservationID = reserveOutput.ReservationID

	// Step 3: Charge payment
	chargeInput := payment.ChargePaymentInput{
		OrderID:       order.ID,
		Amount:        order.TotalAmount,
		PaymentMethod: domain.PaymentMethodCard,
		CustomerID:    order.CustomerID,
	}
	chargeInputBytes, _ := json.Marshal(chargeInput)

	chargeResult := ctx.CallActivity("payment:charge", task.WithActivityInput(chargeInputBytes))
	var chargeOutput payment.ChargePaymentOutput
	if err := chargeResult.Await(&chargeOutput); err != nil {
		// Payment failed - compensate by releasing inventory
		releaseInput := inventory.ReleaseInventoryInput{
			ReservationID: output.ReservationID,
		}
		releaseInputBytes, _ := json.Marshal(releaseInput)
		ctx.CallActivity("inventory:release", task.WithActivityInput(releaseInputBytes)).Await(nil)

		output.Status = "failed"
		output.Message = fmt.Sprintf("payment processing failed: %v", err)
		return marshalOutput(&output)
	}

	output.PaymentID = chargeOutput.PaymentID

	// Step 4: Send confirmation email
	emailInput := notification.EmailNotificationInput{
		CustomerEmail: inp.CustomerEmail,
		OrderID:       order.ID,
		EventType:     "order_confirmed",
	}
	emailInputBytes, _ := json.Marshal(emailInput)

	emailResult := ctx.CallActivity("notification:order_confirmation", task.WithActivityInput(emailInputBytes))
	var emailOutput notification.EmailNotificationOutput
	if err := emailResult.Await(&emailOutput); err != nil {
		// Email failure is non-critical, log but continue
		// In production, you might retry or log to a dead letter queue
	}

	// Success!
	output.Status = "confirmed"
	output.Message = "order processed successfully"

	return marshalOutput(&output)
}

// marshalOutput marshals the output struct to JSON
func marshalOutput(output *OrderProcessingOutput) ([]byte, error) {
	result, err := json.Marshal(output)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal output: %w", err)
	}
	return result, nil
}
