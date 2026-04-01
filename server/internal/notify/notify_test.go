package notify

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
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
	notifier.GeofenceEnter(ctx, "Alice", "Home", []string{"user-1"})

	if len(sender.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(sender.messages))
	}
	msg := sender.messages[0]
	if msg.Token != "user-1" {
		t.Errorf("expected token 'user-1', got %q", msg.Token)
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
	notifier.GeofenceLeave(ctx, "Alice", "Work", []string{"user-1", "user-2"})

	if len(sender.messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(sender.messages))
	}
	expectedBody := "Alice left Work"
	for i, msg := range sender.messages {
		if msg.Body != expectedBody {
			t.Errorf("message %d: expected body %q, got %q", i, expectedBody, msg.Body)
		}
	}
	if sender.messages[0].Token != "user-1" {
		t.Errorf("expected first token 'user-1', got %q", sender.messages[0].Token)
	}
	if sender.messages[1].Token != "user-2" {
		t.Errorf("expected second token 'user-2', got %q", sender.messages[1].Token)
	}
}

func TestNtfySender(t *testing.T) {
	var received []struct {
		topic string
		title string
		body  string
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received = append(received, struct {
			topic string
			title string
			body  string
		}{
			topic: r.URL.Path,
			title: r.Header.Get("Title"),
			body:  string(body),
		})
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	sender, err := NewNtfySender(ts.URL)
	if err != nil {
		t.Fatalf("NewNtfySender: %v", err)
	}

	ctx := context.Background()
	err = sender.Send(ctx, Message{Token: "user-123", Title: "Place Alert", Body: "Alice arrived"})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	if len(received) != 1 {
		t.Fatalf("expected 1 request, got %d", len(received))
	}
	if received[0].topic != "/tracker-user-123" {
		t.Errorf("topic = %q, want /tracker-user-123", received[0].topic)
	}
	if received[0].title != "Place Alert" {
		t.Errorf("title = %q, want Place Alert", received[0].title)
	}
	if received[0].body != "Alice arrived" {
		t.Errorf("body = %q, want Alice arrived", received[0].body)
	}
}
