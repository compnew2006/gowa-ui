package whatsmeow

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/zerodha/logf"
	"go.mau.fi/whatsmeow"
	"gorm.io/gorm"
)

const defaultAdapterReconnectTimeout = 15 * time.Second

// WhatsmeowAdapter implements the MessageProvider interface using whatsmeow.
type WhatsmeowAdapter struct {
	manager          *ConnectionManager
	logger           logf.Logger
	db               *gorm.DB
	getRuntimeClient func(uuid.UUID) *whatsmeow.Client
	connectRuntime   func(context.Context, uuid.UUID) error
	reconnectTimeout time.Duration
}

// NewWhatsmeowAdapter creates a new WhatsmeowAdapter.
func NewWhatsmeowAdapter(manager *ConnectionManager, db *gorm.DB, logger logf.Logger) *WhatsmeowAdapter {
	var getRuntimeClient func(uuid.UUID) *whatsmeow.Client
	var connectRuntime func(context.Context, uuid.UUID) error
	if manager != nil {
		getRuntimeClient = manager.GetClient
		connectRuntime = manager.Connect
	}

	return &WhatsmeowAdapter{
		manager:          manager,
		logger:           logger,
		db:               db,
		getRuntimeClient: getRuntimeClient,
		connectRuntime:   connectRuntime,
		reconnectTimeout: defaultAdapterReconnectTimeout,
	}
}
