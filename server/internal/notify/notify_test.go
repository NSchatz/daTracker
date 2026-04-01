package notify

import (
	"context"
	"testing"
)

type mockSender struct {
	messages []Message
}

func (m *mockSender) Send(ctx context.Context, msg Message) error {
	m.messages = append(m.messages, msg)
	return nil
}

func TestGeofenceEnter(t *testing.T) {
	sender := &mockSender{}
	notifier := NewNotifier(sender)

	ctx := context.Background()
	notifier.GeofenceEnter(ctx, "Alice", "Home", []string{"token-1"})

	if len(sender.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(sender.messages))
	}
	msg := sender.messages[0]
	if msg.Token != "token-1" {
		t.Errorf("expected token 'token-1', got %q", msg.Token)
	}
	expectedBody := "Alice arrived at Home"
	if msg.Body != expectedBody {
		t.Errorf("expected body %q, got %q", expectedBody, msg.Body)
	}
}

func TestGeofenceLeave(t *testing.T) {
	sender := &mockSender{}
	notifier := NewNotifier(sender)

	ctx := context.Background()
	notifier.GeofenceLeave(ctx, "Alice", "Work", []string{"token-1", "token-2"})

	if len(sender.messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(sender.messages))
	}
	expectedBody := "Alice left Work"
	for i, msg := range sender.messages {
		if msg.Body != expectedBody {
			t.Errorf("message %d: expected body %q, got %q", i, expectedBody, msg.Body)
		}
	}
	if sender.messages[0].Token != "token-1" {
		t.Errorf("expected first token 'token-1', got %q", sender.messages[0].Token)
	}
	if sender.messages[1].Token != "token-2" {
		t.Errorf("expected second token 'token-2', got %q", sender.messages[1].Token)
	}
}
