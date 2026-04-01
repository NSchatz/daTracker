package notify

import (
	"context"
	"fmt"
	"log"
)

type Message struct {
	Token string
	Title string
	Body  string
}

type Sender interface {
	Send(ctx context.Context, msg Message) error
}

type Notifier struct {
	sender Sender
}

func NewNotifier(sender Sender) *Notifier {
	return &Notifier{sender: sender}
}

func (n *Notifier) GeofenceEnter(ctx context.Context, userName, placeName string, userIDs []string) {
	body := fmt.Sprintf("%s arrived at %s", userName, placeName)
	for _, userID := range userIDs {
		msg := Message{
			Token: userID,
			Title: "Place Alert",
			Body:  body,
		}
		if err := n.sender.Send(ctx, msg); err != nil {
			log.Printf("notify: GeofenceEnter send error for user %s: %v", userID, err)
		}
	}
}

func (n *Notifier) GeofenceLeave(ctx context.Context, userName, placeName string, userIDs []string) {
	body := fmt.Sprintf("%s left %s", userName, placeName)
	for _, userID := range userIDs {
		msg := Message{
			Token: userID,
			Title: "Place Alert",
			Body:  body,
		}
		if err := n.sender.Send(ctx, msg); err != nil {
			log.Printf("notify: GeofenceLeave send error for user %s: %v", userID, err)
		}
	}
}
