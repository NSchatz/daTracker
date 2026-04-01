package notify

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type NtfySender struct {
	baseURL string
	client  *http.Client
}

func NewNtfySender(baseURL string) (*NtfySender, error) {
	return &NtfySender{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{},
	}, nil
}

func (n *NtfySender) Send(ctx context.Context, msg Message) error {
	topic := fmt.Sprintf("tracker-%s", msg.Token)
	url := fmt.Sprintf("%s/%s", n.baseURL, topic)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(msg.Body))
	if err != nil {
		return fmt.Errorf("ntfy: create request: %w", err)
	}
	req.Header.Set("Title", msg.Title)
	req.Header.Set("Priority", "high")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("ntfy: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ntfy: unexpected status %d", resp.StatusCode)
	}
	return nil
}

type NoopSender struct{}

func (n NoopSender) Send(ctx context.Context, msg Message) error {
	fmt.Printf("[noop-ntfy] topic=tracker-%s title=%q body=%q\n", msg.Token, msg.Title, msg.Body)
	return nil
}
