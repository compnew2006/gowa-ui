package assignment

import (
	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/models"
	"gorm.io/gorm"
)

// agentLoad is a helper struct for scanning load counts.
type agentLoad struct {
	AgentID uuid.UUID `gorm:"column:agent_id"`
	Count   int64     `gorm:"column:count"`
}

// ChatLoadCounter counts active AgentTransfer records per agent.
func ChatLoadCounter(db *gorm.DB, orgID uuid.UUID, agentIDs []uuid.UUID) map[uuid.UUID]int64 {
	var loads []agentLoad
	db.Model(&models.AgentTransfer{}).
		Select("agent_id, COUNT(*) as count").
		Where("organization_id = ? AND agent_id IN ? AND status = ?", orgID, agentIDs, models.TransferStatusActive).
		Group("agent_id").
		Scan(&loads)

	loadMap := make(map[uuid.UUID]int64, len(loads))
	for _, l := range loads {
		loadMap[l.AgentID] = l.Count
	}
	return loadMap
}
