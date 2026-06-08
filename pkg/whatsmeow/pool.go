package whatsmeow

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"go.mau.fi/whatsmeow"
)

// InstanceKey identifies a tenant-owned WhatsApp runtime connection.
type InstanceKey struct {
	OrganizationID uuid.UUID
	AccountName    string
}

// NewInstanceKey normalizes the account name used by the runtime connection pool.
func NewInstanceKey(orgID uuid.UUID, accountName string) InstanceKey {
	return InstanceKey{
		OrganizationID: orgID,
		AccountName:    strings.TrimSpace(accountName),
	}
}

// ConnectionSnapshot captures the observable state for one runtime connection entry.
type ConnectionSnapshot struct {
	InstanceID           uuid.UUID
	Key                  InstanceKey
	OrgID                uuid.UUID
	PhoneNumber          string
	Client               *whatsmeow.Client
	LastDisconnectAt     time.Time
	LastReconnectAttempt time.Time
	ConsecutiveFailures  int
	Reconnecting         bool
	Disabled             bool
}

// ConnectionPool stores active WhatsApp runtime clients keyed by tenant/account identity.
//
// Locking strategy:
//   - sync.Map (byKey, byInstanceID): lock-free reads in hot paths (GetClient, snapshot).
//   - sync.Mutex (mu): serializes structural mutations (ensureEntry, reindex, removeInstance)
//     to prevent duplicate entries during concurrent Connect/Disconnect calls.
//   Do not hold mu while performing I/O or calling external methods.
type ConnectionPool struct {
	byKey        sync.Map // map[InstanceKey]*connectionEntry
	byInstanceID sync.Map // map[uuid.UUID]*connectionEntry
	mu           sync.Mutex
}

type connectionEntry struct {
	connectMu sync.Mutex
	stateMu   sync.RWMutex

	InstanceID uuid.UUID
	Key        InstanceKey
	OrgID      uuid.UUID

	PhoneNumber string
	Client      *whatsmeow.Client

	lastDisconnectAt     time.Time
	lastReconnectAttempt time.Time
	consecutiveFailures  int
	reconnecting         bool
	disabled             bool
	removed              bool
}

func newConnectionEntry(instance models.WhatsAppInstance) *connectionEntry {
	return &connectionEntry{
		InstanceID:  instance.ID,
		Key:         NewInstanceKey(instance.OrganizationID, instance.Name),
		OrgID:       instance.OrganizationID,
		PhoneNumber: strings.TrimSpace(instance.PhoneNumber),
	}
}

// NewConnectionPool creates a new thread-safe WhatsApp runtime registry.
func NewConnectionPool() *ConnectionPool {
	return &ConnectionPool{}
}

// GetClient returns the runtime client registered for the instance ID.
func (p *ConnectionPool) GetClient(instanceID uuid.UUID) *whatsmeow.Client {
	if p == nil {
		return nil
	}

	entry := p.entry(instanceID)
	if entry == nil {
		return nil
	}

	return entry.client()
}

// GetClientByKey returns the runtime client registered for the tenant/account key.
func (p *ConnectionPool) GetClientByKey(key InstanceKey) *whatsmeow.Client {
	if p == nil {
		return nil
	}

	normalizedKey := NewInstanceKey(key.OrganizationID, key.AccountName)
	rawEntry, ok := p.byKey.Load(normalizedKey)
	if !ok {
		return nil
	}

	entry, _ := rawEntry.(*connectionEntry)
	if entry == nil {
		return nil
	}

	return entry.client()
}

// RegisterInstanceClient registers or replaces the runtime client for an instance.
func (p *ConnectionPool) RegisterInstanceClient(instance models.WhatsAppInstance, client *whatsmeow.Client) error {
	if p == nil {
		return fmt.Errorf("connection pool is not initialized")
	}

	entry, conflictID, err := p.ensureEntry(instance)
	if err != nil {
		return err
	}
	if conflictID != uuid.Nil {
		return fmt.Errorf("instance key %q for organization %s is already owned by instance %s", strings.TrimSpace(instance.Name), instance.OrganizationID, conflictID)
	}

	entry.attachClient(client, instance.PhoneNumber)
	return nil
}

// ReindexInstance moves a connected instance to its latest tenant/account key.
func (p *ConnectionPool) ReindexInstance(instance models.WhatsAppInstance) error {
	if p == nil {
		return fmt.Errorf("connection pool is not initialized")
	}

	_, conflictID, err := p.reindexInstance(instance)
	if err != nil {
		return err
	}
	if conflictID != uuid.Nil {
		return fmt.Errorf("instance key %q for organization %s is already owned by instance %s", strings.TrimSpace(instance.Name), instance.OrganizationID, conflictID)
	}

	return nil
}

func (p *ConnectionPool) removeInstance(instanceID uuid.UUID) *whatsmeow.Client {
	if p == nil {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	rawEntry, ok := p.byInstanceID.Load(instanceID)
	if !ok {
		return nil
	}

	entry, _ := rawEntry.(*connectionEntry)
	if entry == nil {
		p.byInstanceID.Delete(instanceID)
		return nil
	}

	currentKey := entry.key()
	p.byInstanceID.Delete(instanceID)
	p.byKey.CompareAndDelete(currentKey, entry)
	entry.markRemoved()
	return entry.client()
}

// SnapshotEntries returns a copy of the runtime pool state for observability and tests.
func (p *ConnectionPool) SnapshotEntries() []ConnectionSnapshot {
	if p == nil {
		return nil
	}

	entries := p.snapshotEntries()
	snapshots := make([]ConnectionSnapshot, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		snapshots = append(snapshots, entry.snapshot())
	}

	return snapshots
}

func (p *ConnectionPool) entry(instanceID uuid.UUID) *connectionEntry {
	if p == nil {
		return nil
	}

	rawEntry, ok := p.byInstanceID.Load(instanceID)
	if !ok {
		return nil
	}

	entry, _ := rawEntry.(*connectionEntry)
	return entry
}

func (p *ConnectionPool) ensureEntry(instance models.WhatsAppInstance) (*connectionEntry, uuid.UUID, error) {
	if p == nil {
		return nil, uuid.Nil, fmt.Errorf("connection pool is not initialized")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	return p.ensureEntryLocked(instance)
}

func (p *ConnectionPool) ensureEntryLocked(instance models.WhatsAppInstance) (*connectionEntry, uuid.UUID, error) {
	key := NewInstanceKey(instance.OrganizationID, instance.Name)
	if key.AccountName == "" {
		return nil, uuid.Nil, fmt.Errorf("instance %s has empty account name", instance.ID)
	}

	if rawEntry, ok := p.byInstanceID.Load(instance.ID); ok {
		entry, _ := rawEntry.(*connectionEntry)
		if entry == nil {
			p.byInstanceID.Delete(instance.ID)
		} else {
			if rawConflict, exists := p.byKey.Load(key); exists {
				conflict, _ := rawConflict.(*connectionEntry)
				if conflict != nil && conflict.InstanceID != instance.ID {
					return nil, conflict.InstanceID, nil
				}
			}

			oldKey := entry.key()
			entry.updateMetadata(key, strings.TrimSpace(instance.PhoneNumber))
			p.byInstanceID.Store(instance.ID, entry)
			p.byKey.Store(key, entry)
			if oldKey != key {
				p.byKey.CompareAndDelete(oldKey, entry)
			}
			return entry, uuid.Nil, nil
		}
	}

	if rawConflict, ok := p.byKey.Load(key); ok {
		conflict, _ := rawConflict.(*connectionEntry)
		if conflict != nil && conflict.InstanceID != instance.ID {
			return nil, conflict.InstanceID, nil
		}
	}

	entry := newConnectionEntry(instance)
	p.byInstanceID.Store(instance.ID, entry)
	p.byKey.Store(key, entry)
	return entry, uuid.Nil, nil
}

func (p *ConnectionPool) reindexInstance(instance models.WhatsAppInstance) (*connectionEntry, uuid.UUID, error) {
	if p == nil {
		return nil, uuid.Nil, fmt.Errorf("connection pool is not initialized")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	rawEntry, ok := p.byInstanceID.Load(instance.ID)
	if !ok {
		return nil, uuid.Nil, nil
	}

	entry, _ := rawEntry.(*connectionEntry)
	if entry == nil {
		p.byInstanceID.Delete(instance.ID)
		return nil, uuid.Nil, nil
	}

	return p.ensureEntryLocked(instance)
}

func (p *ConnectionPool) markConnected(instanceID uuid.UUID, phoneNumber string) {
	entry := p.entry(instanceID)
	if entry == nil {
		return
	}

	entry.markConnected(phoneNumber)
}

func (p *ConnectionPool) markDisconnected(instanceID uuid.UUID) {
	entry := p.entry(instanceID)
	if entry == nil {
		return
	}

	entry.markDisconnected()
}

func (p *ConnectionPool) snapshotEntries() []*connectionEntry {
	if p == nil {
		return nil
	}

	entries := make([]*connectionEntry, 0)
	p.byInstanceID.Range(func(_, value any) bool {
		entry, _ := value.(*connectionEntry)
		if entry != nil {
			entries = append(entries, entry)
		}
		return true
	})
	return entries
}

func (e *connectionEntry) client() *whatsmeow.Client {
	if e == nil {
		return nil
	}

	e.stateMu.RLock()
	defer e.stateMu.RUnlock()
	return e.Client
}

func (e *connectionEntry) key() InstanceKey {
	if e == nil {
		return InstanceKey{}
	}

	e.stateMu.RLock()
	defer e.stateMu.RUnlock()
	return e.Key
}

func (e *connectionEntry) updateMetadata(key InstanceKey, phoneNumber string) {
	if e == nil {
		return
	}

	e.stateMu.Lock()
	defer e.stateMu.Unlock()
	e.Key = key
	e.OrgID = key.OrganizationID
	if phoneNumber != "" {
		e.PhoneNumber = phoneNumber
	}
	e.removed = false
}

func (e *connectionEntry) attachClient(client *whatsmeow.Client, phoneNumber string) {
	if e == nil {
		return
	}

	e.stateMu.Lock()
	defer e.stateMu.Unlock()
	e.Client = client
	if trimmedPhone := strings.TrimSpace(phoneNumber); trimmedPhone != "" {
		e.PhoneNumber = trimmedPhone
	}
	e.removed = false
}

func (e *connectionEntry) markConnected(phoneNumber string) {
	if e == nil {
		return
	}

	e.stateMu.Lock()
	defer e.stateMu.Unlock()
	if trimmedPhone := strings.TrimSpace(phoneNumber); trimmedPhone != "" {
		e.PhoneNumber = trimmedPhone
	}
	e.consecutiveFailures = 0
	e.reconnecting = false
	e.disabled = false
	e.removed = false
}

func (e *connectionEntry) markDisconnected() {
	if e == nil {
		return
	}

	e.stateMu.Lock()
	defer e.stateMu.Unlock()
	e.lastDisconnectAt = time.Now().UTC()
	e.reconnecting = false
}

func (e *connectionEntry) markRemoved() {
	if e == nil {
		return
	}

	e.stateMu.Lock()
	defer e.stateMu.Unlock()
	e.removed = true
	e.reconnecting = false
}

func (e *connectionEntry) beginReconnect(now time.Time, baseDelay, maxDelay time.Duration) bool {
	if e == nil {
		return false
	}

	e.stateMu.Lock()
	defer e.stateMu.Unlock()

	if e.removed || e.disabled || e.reconnecting {
		return false
	}

	if !e.lastReconnectAttempt.IsZero() {
		delay := baseDelay
		if e.consecutiveFailures > 1 {
			delay = baseDelay << minInt(e.consecutiveFailures-1, 6)
		}
		if delay > maxDelay {
			delay = maxDelay
		}
		if now.Sub(e.lastReconnectAttempt) < delay {
			return false
		}
	}

	e.reconnecting = true
	e.lastReconnectAttempt = now
	return true
}

func (e *connectionEntry) finishReconnect(err error, phoneNumber string) {
	if e == nil {
		return
	}

	e.stateMu.Lock()
	defer e.stateMu.Unlock()
	e.reconnecting = false
	if err != nil {
		e.consecutiveFailures++
		return
	}

	if trimmedPhone := strings.TrimSpace(phoneNumber); trimmedPhone != "" {
		e.PhoneNumber = trimmedPhone
	}
	e.consecutiveFailures = 0
	e.disabled = false
	e.removed = false
}

func (e *connectionEntry) snapshot() ConnectionSnapshot {
	if e == nil {
		return ConnectionSnapshot{}
	}

	e.stateMu.RLock()
	defer e.stateMu.RUnlock()

	return ConnectionSnapshot{
		InstanceID:           e.InstanceID,
		Key:                  e.Key,
		OrgID:                e.OrgID,
		PhoneNumber:          e.PhoneNumber,
		Client:               e.Client,
		LastDisconnectAt:     e.lastDisconnectAt,
		LastReconnectAttempt: e.lastReconnectAttempt,
		ConsecutiveFailures:  e.consecutiveFailures,
		Reconnecting:         e.reconnecting,
		Disabled:             e.disabled,
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
