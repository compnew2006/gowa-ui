package websocket

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/zerodha/logf"
)

// Hub maintains the set of active clients and broadcasts messages to them
type Hub struct {
	// clients maps organization ID -> user ID -> set of clients (supports multiple tabs)
	clients map[uuid.UUID]map[uuid.UUID]map[*Client]struct{}

	// broadcast channel for messages
	broadcast chan BroadcastMessage

	// register channel for new clients
	register chan *Client

	// unregister channel for disconnecting clients
	unregister chan *Client

	// mutex for thread-safe access to clients map
	mu sync.RWMutex

	// logger
	log logf.Logger

	// Redis client for Pub/Sub
	rdb *redis.Client

	// Redis PubSub connection
	pubsub *redis.PubSub

	// Unique ID of this hub instance to prevent self-echo loops
	instanceID uuid.UUID

	// Channel for triggering subscription state reconciliation
	reconcileChan chan struct{}
}

// NewHub creates a new Hub instance
func NewHub(log logf.Logger, rdb *redis.Client) *Hub {
	var pubsub *redis.PubSub
	if rdb != nil {
		pubsub = rdb.Subscribe(context.Background())
	}
	return &Hub{
		clients:       make(map[uuid.UUID]map[uuid.UUID]map[*Client]struct{}),
		broadcast:     make(chan BroadcastMessage, 256),
		register:      make(chan *Client, 256),
		unregister:    make(chan *Client, 256),
		log:           log,
		rdb:           rdb,
		pubsub:        pubsub,
		instanceID:    uuid.New(),
		reconcileChan: make(chan struct{}, 1),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	if h.pubsub != nil {
		go h.manageSubscriptions()
		go h.listenRedisBroadcasts()
	}

	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

// manageSubscriptions reconciles subscription state based on notifications and periodic tick fallback.
func (h *Hub) manageSubscriptions() {
	subscribed := make(map[string]bool)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Initial reconciliation on startup in case clients registered before Run was active
	h.reconcile(subscribed)

	for {
		select {
		case <-h.reconcileChan:
			h.reconcile(subscribed)
		case <-ticker.C:
			h.reconcile(subscribed)
		}
	}
}

// triggerReconcile non-blockingly signals a subscription reconciliation
func (h *Hub) triggerReconcile() {
	if h.pubsub != nil {
		select {
		case h.reconcileChan <- struct{}{}:
		default:
			// Reconcile is already scheduled, which is sufficient
		}
	}
}

// getActiveOrgs safely retrieves the current active organization IDs
func (h *Hub) getActiveOrgs() []uuid.UUID {
	h.mu.RLock()
	defer h.mu.RUnlock()

	orgs := make([]uuid.UUID, 0, len(h.clients))
	for orgID := range h.clients {
		orgs = append(orgs, orgID)
	}
	return orgs
}

// reconcile compares the desired channels against currently subscribed channels
// and performs necessary Subscribe/Unsubscribe operations.
func (h *Hub) reconcile(subscribed map[string]bool) {
	activeOrgs := h.getActiveOrgs()
	desired := make(map[string]bool, len(activeOrgs))
	for _, orgID := range activeOrgs {
		channel := "whatomate:ws_broadcast:org:" + orgID.String()
		desired[channel] = true
	}

	// 1. Subscribe to channels that are desired but not subscribed yet
	for channel := range desired {
		if !subscribed[channel] {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := h.pubsub.Subscribe(ctx, channel)
			cancel()
			if err == nil {
				subscribed[channel] = true
				h.log.Info("Subscribed to Redis org channel", "channel", channel)
			} else {
				h.log.Error("Failed to subscribe to Redis org channel, will retry",
					"channel", channel,
					"error", err)
			}
		}
	}

	// 2. Unsubscribe from channels that are subscribed but no longer desired
	for channel := range subscribed {
		if !desired[channel] {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := h.pubsub.Unsubscribe(ctx, channel)
			cancel()
			if err == nil {
				delete(subscribed, channel)
				h.log.Info("Unsubscribed from Redis org channel", "channel", channel)
			} else {
				h.log.Error("Failed to unsubscribe from Redis org channel, will retry",
					"channel", channel,
					"error", err)
			}
		}
	}
}

// listenRedisBroadcasts subscribes to the Redis Pub/Sub channel and routes messages locally.
func (h *Hub) listenRedisBroadcasts() {
	defer h.pubsub.Close()

	ch := h.pubsub.Channel()
	for msg := range ch {
		var broadcastMsg BroadcastMessage
		if err := json.Unmarshal([]byte(msg.Payload), &broadcastMsg); err != nil {
			h.log.Error("Failed to unmarshal Redis broadcast message", "error", err)
			continue
		}

		// Verify that the message was received on the correct channel matching its OrgID to prevent cross-tenant leakage
		channelPrefix := "whatomate:ws_broadcast:org:"
		if strings.HasPrefix(msg.Channel, channelPrefix) {
			expectedOrgID := strings.TrimPrefix(msg.Channel, channelPrefix)
			if broadcastMsg.OrgID.String() != expectedOrgID {
				h.log.Warn("Mismatch between Redis channel org ID and payload org ID, dropping message",
					"channel", msg.Channel,
					"payload_org", broadcastMsg.OrgID)
				continue
			}
		}

		// Suppress self-echo
		if broadcastMsg.SenderInstanceID == h.instanceID {
			continue
		}

		select {
		case h.broadcast <- broadcastMsg:
		default:
			h.log.Warn("Local broadcast channel full, dropping message from Redis")
		}
	}
}

// registerClient adds a client to the hub
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	orgClients, ok := h.clients[client.organizationID]
	if !ok {
		orgClients = make(map[uuid.UUID]map[*Client]struct{})
		h.clients[client.organizationID] = orgClients

		// Trigger subscription state reconciliation
		h.triggerReconcile()
	}

	userClients, ok := orgClients[client.userID]
	if !ok {
		userClients = make(map[*Client]struct{})
		orgClients[client.userID] = userClients
	}

	// Add this client to the set (allows multiple tabs)
	userClients[client] = struct{}{}

	h.log.Info("WebSocket client registered",
		"user_id", client.userID,
		"org_id", client.organizationID,
		"user_connections", len(userClients),
		"total_clients", h.countClients())
}

// unregisterClient removes a client from the hub
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if orgClients, ok := h.clients[client.organizationID]; ok {
		if userClients, ok := orgClients[client.userID]; ok {
			if _, exists := userClients[client]; exists {
				delete(userClients, client)
				close(client.send)

				// Clean up empty user map
				if len(userClients) == 0 {
					delete(orgClients, client.userID)
				}

				// Clean up empty org map
				if len(orgClients) == 0 {
					delete(h.clients, client.organizationID)

					// Trigger subscription state reconciliation
					h.triggerReconcile()
				}
			}
		}
	}

	h.log.Info("WebSocket client unregistered",
		"user_id", client.userID,
		"org_id", client.organizationID,
		"total_clients", h.countClients())
}

// broadcastMessage sends a message to all relevant clients
func (h *Hub) broadcastMessage(msg BroadcastMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	orgClients, ok := h.clients[msg.OrgID]
	if !ok {
		return
	}

	data, err := json.Marshal(msg.Message)
	if err != nil {
		h.log.Error("Failed to marshal broadcast message", "error", err)
		return
	}

	// If UserID is specified, only send to that user's clients
	if msg.UserID != uuid.Nil {
		userClients, ok := orgClients[msg.UserID]
		if !ok {
			return
		}
		for client := range userClients {
			select {
			case client.send <- data:
			default:
				h.log.Warn("Client send buffer full, skipping",
					"user_id", client.userID,
					"org_id", client.organizationID)
			}
		}
		return
	}

	// Iterate through all users in the organization
	for _, userClients := range orgClients {
		// Iterate through all clients (tabs) for each user
		for client := range userClients {
			// If ContactID is specified, only send to clients explicitly subscribed
			// to that contact.
			if msg.ContactID != uuid.Nil {
				currentContact := client.getCurrentContact()
				if currentContact == nil || *currentContact != msg.ContactID {
					continue
				}
			}

			select {
			case client.send <- data:
			default:
				// Client buffer full, skip
				h.log.Warn("Client send buffer full, skipping",
					"user_id", client.userID,
					"org_id", client.organizationID)
			}
		}
	}
}

// Broadcast sends a message to the broadcast channel
func (h *Hub) Broadcast(msg BroadcastMessage) {
	// Deliver locally immediately
	select {
	case h.broadcast <- msg:
	default:
		h.log.Warn("Broadcast channel full, dropping local message")
	}

	// Publish to Redis for cross-instance propagation
	if h.rdb != nil {
		msg.SenderInstanceID = h.instanceID
		ctx := context.Background()
		data, err := json.Marshal(msg)
		if err != nil {
			h.log.Error("Failed to marshal broadcast message for Redis", "error", err)
			return
		}
		channel := "whatomate:ws_broadcast:org:" + msg.OrgID.String()
		if err := h.rdb.Publish(ctx, channel, data).Err(); err != nil {
			h.log.Error("Failed to publish broadcast message to Redis", "error", err)
		}
	}
}

// BroadcastToOrg sends a message to all clients in an organization
func (h *Hub) BroadcastToOrg(orgID uuid.UUID, msg WSMessage) {
	h.Broadcast(BroadcastMessage{
		OrgID:   orgID,
		Message: msg,
	})
}

// BroadcastToContact sends a message to clients viewing a specific contact
func (h *Hub) BroadcastToContact(orgID, contactID uuid.UUID, msg WSMessage) {
	h.Broadcast(BroadcastMessage{
		OrgID:     orgID,
		ContactID: contactID,
		Message:   msg,
	})
}

// BroadcastToUser sends a message to a specific user
func (h *Hub) BroadcastToUser(orgID, userID uuid.UUID, msg WSMessage) {
	h.Broadcast(BroadcastMessage{
		OrgID:   orgID,
		UserID:  userID,
		Message: msg,
	})
}

// BroadcastToUsers sends a message to multiple users
func (h *Hub) BroadcastToUsers(orgID uuid.UUID, userIDs []uuid.UUID, msg WSMessage) {
	for _, userID := range userIDs {
		h.BroadcastToUser(orgID, userID, msg)
	}
}

// countClients returns the total number of connected clients
func (h *Hub) countClients() int {
	count := 0
	for _, orgClients := range h.clients {
		for _, userClients := range orgClients {
			count += len(userClients)
		}
	}
	return count
}

// GetClientCount returns the number of connected clients (thread-safe)
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.countClients()
}

// Register adds a client to the hub via the register channel
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client from the hub via the unregister channel
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}
