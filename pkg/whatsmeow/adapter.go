package whatsmeow

import (
	"github.com/zerodha/logf"
	"gorm.io/gorm"
)

// WhatsmeowAdapter implements the MessageProvider interface using whatsmeow.
type WhatsmeowAdapter struct {
	manager *ConnectionManager
	logger  logf.Logger
	db      *gorm.DB
}

// NewWhatsmeowAdapter creates a new WhatsmeowAdapter.
func NewWhatsmeowAdapter(manager *ConnectionManager, db *gorm.DB, logger logf.Logger) *WhatsmeowAdapter {
	return &WhatsmeowAdapter{
		manager: manager,
		logger:  logger,
		db:      db,
	}
}
