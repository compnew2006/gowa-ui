package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOrganizationConfigTableName(t *testing.T) {
	assert.Equal(t, "organization_configs", OrganizationConfig{}.TableName())
}

func TestOrganizationConfigDefaults(t *testing.T) {
	cfg := OrganizationConfig{}
	assert.Equal(t, 0, cfg.WorkerCount)
	assert.Equal(t, 0, cfg.MaxQueueSize)
	assert.Equal(t, 0, cfg.MaxWhatsAppInstances)
	assert.Nil(t, cfg.Organization)
}
