package app

import (
	"context"

	"github.com/shellhaki/spark-whatsapp-module/internal/config"
	"github.com/shellhaki/spark-whatsapp-module/internal/messages"
	"github.com/shellhaki/spark-whatsapp-module/internal/subscribers"
)

type Sender interface {
	SendToSubscribers(ctx context.Context, request messages.DeliveryRequest) ([]messages.DeliveryResult, error)
	SendToPhoneNumber(ctx context.Context, request messages.DirectDeliveryRequest) (messages.DeliveryResult, error)
}

type App struct {
	Config      config.Config
	Subscribers *subscribers.Repository
	Sender      Sender
}

func New(cfg config.Config, repo *subscribers.Repository, sender Sender) *App {
	return &App{
		Config:      cfg,
		Subscribers: repo,
		Sender:      sender,
	}
}
