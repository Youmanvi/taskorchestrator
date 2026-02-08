package notification

import (
	"context"
	"fmt"
	"sync"
)

// MockEmailService is a mock implementation of EmailService for testing
type MockEmailService struct {
	mu       sync.RWMutex
	messages map[string]*EmailMessage
	counter  int
}

// EmailMessage represents a sent email
type EmailMessage struct {
	ID      string
	To      string
	Subject string
	Body    string
}

// NewMockEmailService creates a new mock email service
func NewMockEmailService() *MockEmailService {
	return &MockEmailService{
		messages: make(map[string]*EmailMessage),
	}
}

// SendEmail simulates sending an email
func (m *MockEmailService) SendEmail(ctx context.Context, to, subject, body string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.counter++
	messageID := fmt.Sprintf("MSG_%d", m.counter)

	m.messages[messageID] = &EmailMessage{
		ID:      messageID,
		To:      to,
		Subject: subject,
		Body:    body,
	}

	return messageID, nil
}

// GetMessage retrieves a sent message
func (m *MockEmailService) GetMessage(messageID string) (*EmailMessage, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	msg, exists := m.messages[messageID]
	return msg, exists
}

// GetAllMessages returns all sent messages
func (m *MockEmailService) GetAllMessages() []*EmailMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()

	messages := make([]*EmailMessage, 0, len(m.messages))
	for _, msg := range m.messages {
		messages = append(messages, msg)
	}
	return messages
}
