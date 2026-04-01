package notify

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// FCMSender uses Firebase Admin SDK.
type FCMSender struct {
	client *messaging.Client
}

func NewFCMSender(ctx context.Context, credentialsFile string) (*FCMSender, error) {
	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		return nil, fmt.Errorf("fcm: firebase.NewApp: %w", err)
	}
	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("fcm: app.Messaging: %w", err)
	}
	return &FCMSender{client: client}, nil
}

func (f *FCMSender) Send(ctx context.Context, msg Message) error {
	_, err := f.client.Send(ctx, &messaging.Message{
		Token: msg.Token,
		Notification: &messaging.Notification{
			Title: msg.Title,
			Body:  msg.Body,
		},
		Android: &messaging.AndroidConfig{
			Priority: "high",
			Notification: &messaging.AndroidNotification{
				ChannelID: "place_alerts",
			},
		},
	})
	if err != nil {
		return fmt.Errorf("fcm: send: %w", err)
	}
	return nil
}

// NoopSender is used when FCM credentials are not configured.
type NoopSender struct{}

func (n NoopSender) Send(ctx context.Context, msg Message) error {
	fmt.Printf("noop fcm: to=%s title=%q body=%q\n", msg.Token, msg.Title, msg.Body)
	return nil
}
