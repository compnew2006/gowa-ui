package whatsmeow

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	waClient "go.mau.fi/whatsmeow"
)

func TestConnectionPool_RegisterGetAndRemove(t *testing.T) {
	t.Parallel()

	pool := NewConnectionPool()
	orgID := uuid.New()
	instance := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: orgID,
		Name:           " Sales ",
		PhoneNumber:    "15550001111",
	}
	client := &waClient.Client{}

	require.NoError(t, pool.RegisterInstanceClient(instance, client))

	key := NewInstanceKey(orgID, instance.Name)
	assert.Same(t, client, pool.GetClient(instance.ID))
	assert.Same(t, client, pool.GetClientByKey(key))

	removedClient := pool.removeInstance(instance.ID)
	assert.Same(t, client, removedClient)
	assert.Nil(t, pool.GetClient(instance.ID))
	assert.Nil(t, pool.GetClientByKey(key))
}

func TestConnectionPool_RegisterRejectsDuplicateKey(t *testing.T) {
	t.Parallel()

	pool := NewConnectionPool()
	orgID := uuid.New()

	first := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: orgID,
		Name:           "Support",
	}
	second := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: orgID,
		Name:           " Support ",
	}

	require.NoError(t, pool.RegisterInstanceClient(first, &waClient.Client{}))
	err := pool.RegisterInstanceClient(second, &waClient.Client{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already owned")
}

func TestConnectionPool_ReindexInstanceMovesKey(t *testing.T) {
	t.Parallel()

	pool := NewConnectionPool()
	orgID := uuid.New()
	instance := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: orgID,
		Name:           "Sales",
	}
	client := &waClient.Client{}

	require.NoError(t, pool.RegisterInstanceClient(instance, client))
	assert.Same(t, client, pool.GetClientByKey(NewInstanceKey(orgID, "Sales")))

	instance.Name = "Growth"
	require.NoError(t, pool.ReindexInstance(instance))

	assert.Nil(t, pool.GetClientByKey(NewInstanceKey(orgID, "Sales")))
	assert.Same(t, client, pool.GetClientByKey(NewInstanceKey(orgID, "Growth")))
	assert.Same(t, client, pool.GetClient(instance.ID))
}

func TestConnectionPool_BeginReconnectSuppressesConcurrentAttempts(t *testing.T) {
	t.Parallel()

	pool := NewConnectionPool()
	instance := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: uuid.New(),
		Name:           "Ops",
	}
	require.NoError(t, pool.RegisterInstanceClient(instance, &waClient.Client{}))

	entry := pool.entry(instance.ID)
	require.NotNil(t, entry)

	var successes int32
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if entry.beginReconnect(time.Now().UTC(), 0, time.Second) {
				atomic.AddInt32(&successes, 1)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(1), successes)
}
