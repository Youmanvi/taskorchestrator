package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vihan/taskorchestrator/internal/workflows"
	"github.com/vihan/taskorchestrator/test/fixtures"
)

func TestOrderProcessingSuccessful(t *testing.T) {
	harness, err := NewTestHarness()
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, harness.Start(ctx))
	defer harness.Stop(ctx)

	// Create order
	order := fixtures.CreateValidOrder()
	input := &workflows.OrderProcessingInput{
		Order:         order,
		CustomerEmail: "customer@example.com",
	}

	// Schedule orchestration
	execution, err := harness.ScheduleOrder(ctx, input)
	require.NoError(t, err)

	// Wait for completion
	result, err := harness.WaitForOrchestration(ctx, execution, 5*time.Second)
	require.NoError(t, err)

	// Verify result
	assert.True(t, result.IsSuccessful())

	output, err := GetOrderOutput(result)
	require.NoError(t, err)

	assert.Equal(t, "confirmed", output.Status)
	assert.Equal(t, order.ID, output.OrderID)
	assert.NotEmpty(t, output.PaymentID)
	assert.NotEmpty(t, output.ReservationID)

	// Verify reservation was created
	_, exists := harness.InventoryMgr.GetReservation(output.ReservationID)
	assert.True(t, exists, "reservation should exist")
}

func TestOrderProcessingSingleItem(t *testing.T) {
	harness, err := NewTestHarness()
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, harness.Start(ctx))
	defer harness.Stop(ctx)

	// Create single item order
	order := fixtures.CreateSingleItemOrder()
	input := &workflows.OrderProcessingInput{
		Order:         order,
		CustomerEmail: "customer@example.com",
	}

	// Schedule and wait
	execution, err := harness.ScheduleOrder(ctx, input)
	require.NoError(t, err)

	result, err := harness.WaitForOrchestration(ctx, execution, 5*time.Second)
	require.NoError(t, err)

	assert.True(t, result.IsSuccessful())

	output, err := GetOrderOutput(result)
	require.NoError(t, err)

	assert.Equal(t, "confirmed", output.Status)
}

func TestOrderProcessingLargeOrder(t *testing.T) {
	harness, err := NewTestHarness()
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, harness.Start(ctx))
	defer harness.Stop(ctx)

	// Create large order
	order := fixtures.CreateLargeOrder()
	input := &workflows.OrderProcessingInput{
		Order:         order,
		CustomerEmail: "customer@example.com",
	}

	// Schedule and wait
	execution, err := harness.ScheduleOrder(ctx, input)
	require.NoError(t, err)

	result, err := harness.WaitForOrchestration(ctx, execution, 5*time.Second)
	require.NoError(t, err)

	assert.True(t, result.IsSuccessful())

	output, err := GetOrderOutput(result)
	require.NoError(t, err)

	assert.Equal(t, "confirmed", output.Status)
	assert.NotEmpty(t, output.PaymentID)
	assert.NotEmpty(t, output.ReservationID)
}

func TestOrderNotifications(t *testing.T) {
	harness, err := NewTestHarness()
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, harness.Start(ctx))
	defer harness.Stop(ctx)

	// Create order
	order := fixtures.CreateValidOrder()
	input := &workflows.OrderProcessingInput{
		Order:         order,
		CustomerEmail: "customer@example.com",
	}

	// Schedule and wait
	execution, err := harness.ScheduleOrder(ctx, input)
	require.NoError(t, err)

	result, err := harness.WaitForOrchestration(ctx, execution, 5*time.Second)
	require.NoError(t, err)

	assert.True(t, result.IsSuccessful())

	// Verify confirmation email was sent
	messages := harness.EmailService.GetAllMessages()
	assert.True(t, len(messages) > 0, "at least one email should be sent")

	// Verify email content
	found := false
	for _, msg := range messages {
		if msg.To == "customer@example.com" {
			found = true
			assert.Contains(t, msg.Subject, order.ID)
			break
		}
	}
	assert.True(t, found, "confirmation email should be sent to customer")
}

func TestMultipleOrdersInParallel(t *testing.T) {
	harness, err := NewTestHarness()
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, harness.Start(ctx))
	defer harness.Stop(ctx)

	// Schedule multiple orders
	numOrders := 3
	type execution interface {
		WaitForCompletion(context.Context) (*api.OrchestrationExecutionResult, error)
	}
	executions := make([]execution, numOrders)

	for i := 0; i < numOrders; i++ {
		order := fixtures.CreateValidOrder()
		input := &workflows.OrderProcessingInput{
			Order:         order,
			CustomerEmail: "customer@example.com",
		}

		exec, err := harness.ScheduleOrder(ctx, input)
		require.NoError(t, err)
		executions[i] = exec
	}

	// Verify all completed
	successCount := 0
	for i := 0; i < numOrders; i++ {
		result, err := harness.WaitForOrchestration(ctx, executions[i], 5*time.Second)
		if err == nil && result.IsSuccessful() {
			successCount++
		}
	}

	assert.Equal(t, numOrders, successCount, "all orders should complete successfully")
}
