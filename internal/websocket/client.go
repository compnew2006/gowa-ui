package websocket

import (
	"encoding/json"
	"net"
	"sync"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/google/uuid"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 4096

	// Time allowed to authenticate after connection
	authTimeout = 5 * time.Second
)

// AuthenticateFn validates a JWT token and returns user ID and organization ID.
type AuthenticateFn func(token string) (uuid.UUID, uuid.UUID, error)

// ContactAccessFn validates that a user may subscribe to a contact-scoped stream.
type ContactAccessFn func(userID, orgID, contactID uuid.UUID) bool

// Client represents a WebSocket client connection
type Client struct {
	hub *Hub

	// The websocket connection
	conn   *websocket.Conn
	connMu sync.Mutex // protects conn from concurrent ReadPump/WritePump access

	// Buffered channel of outbound messages
	send chan []byte

	// User information (set after authentication)
	userID         uuid.UUID
	organizationID uuid.UUID

	// Whether the client has authenticated
	authenticated bool

	// Function to validate JWT tokens
	authFn AuthenticateFn

	// Function to validate contact subscriptions.
	contactAccessFn ContactAccessFn

	// Current contact being viewed (nil if none)
	currentContact *uuid.UUID
}

// NewClient creates a new unauthenticated Client instance.
// The client must authenticate via a message-based auth flow before it can
// send/receive application messages.
func NewClient(hub *Hub, conn *websocket.Conn, userID, orgID uuid.UUID) *Client {
	return &Client{
		hub:            hub,
		conn:           conn,
		send:           make(chan []byte, 256),
		userID:         userID,
		organizationID: orgID,
		authenticated:  userID != uuid.Nil, // pre-authenticated if userID is set (tests)
	}
}

// NewUnauthenticatedClient creates a client that requires message-based authentication.
func NewUnauthenticatedClient(hub *Hub, conn *websocket.Conn, authFn AuthenticateFn) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		authFn: authFn,
	}
}

// SetContactAccessFn configures an optional authorization callback for
// contact-scoped subscriptions.
func (c *Client) SetContactAccessFn(fn ContactAccessFn) {
	c.contactAccessFn = fn
}

// ReadPump pumps messages from the websocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		if r := recover(); r != nil {
			c.hub.log.Error("Recovered from panic in ReadPump", "error", r, "user_id", c.userID)
		}
		if c.authenticated {
			c.hub.unregister <- c
		} else if c.send != nil {
			// Unauthenticated clients are never registered in hub, so close send here
			// to ensure WritePump exits immediately instead of lingering.
			close(c.send)
		}
		// Close conn under lock so WritePump sees conn==nil before it tries to write.
		c.connMu.Lock()
		if c.conn != nil {
			_ = c.conn.Close()
			c.conn = nil
		}
		c.connMu.Unlock()
	}()

	c.conn.SetReadLimit(maxMessageSize)

	// If not yet authenticated, enforce auth timeout for the first message
	if !c.authenticated {
		_ = c.conn.SetReadDeadline(time.Now().Add(authTimeout))

		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				c.hub.log.Debug("WebSocket auth timeout", "timeout", authTimeout.String())
				return
			}
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				c.hub.log.Debug("WebSocket closed before auth", "error", err)
				return
			}
			c.hub.log.Warn("WebSocket auth read error", "error", err)
			return
		}

		if !c.handleAuthMessage(message) {
			_ = c.conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "authentication failed"))
			return
		}
	}

	// Normal read loop (authenticated)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.hub.log.Error("WebSocket read error", "error", err, "user_id", c.userID)
			}
			break
		}

		c.handleMessage(message)
	}
}

// WritePump pumps messages from the hub to the websocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		if r := recover(); r != nil {
			c.hub.log.Error("Recovered from panic in WritePump", "error", r, "user_id", c.userID)
		}
		ticker.Stop()
		c.connMu.Lock()
		if c.conn != nil {
			_ = c.conn.Close()
			c.conn = nil
		}
		c.connMu.Unlock()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.connMu.Lock()
			if c.conn == nil {
				c.connMu.Unlock()
				return
			}
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				c.connMu.Unlock()
				return
			}
			if !ok {
				// Channel closed: reader/hub is done, exit writer.
				c.connMu.Unlock()
				return
			}

			// Only forward messages if authenticated
			if !c.authenticated {
				c.connMu.Unlock()
				continue
			}

			// Send each message as a separate WebSocket frame
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				c.connMu.Unlock()
				return
			}

			// Send any queued messages as separate frames
			n := len(c.send)
			for i := 0; i < n; i++ {
				if err := c.conn.WriteMessage(websocket.TextMessage, <-c.send); err != nil {
					c.connMu.Unlock()
					return
				}
			}
			c.connMu.Unlock()

		case <-ticker.C:
			c.connMu.Lock()
			if c.conn == nil {
				c.connMu.Unlock()
				return
			}
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				c.connMu.Unlock()
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.connMu.Unlock()
				return
			}
			c.connMu.Unlock()
		}
	}
}

// handleAuthMessage processes the first message which must be an auth message.
// Returns true if authentication succeeded.
func (c *Client) handleAuthMessage(data []byte) bool {
	var msg WSMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		c.hub.log.Error("Failed to unmarshal auth message", "error", err)
		return false
	}

	if msg.Type != TypeAuth {
		c.hub.log.Warn("Expected auth message, got", "type", msg.Type)
		return false
	}

	payloadBytes, err := json.Marshal(msg.Payload)
	if err != nil {
		return false
	}

	var authPayload AuthPayload
	if err := json.Unmarshal(payloadBytes, &authPayload); err != nil {
		return false
	}

	if authPayload.Token == "" || c.authFn == nil {
		return false
	}

	userID, orgID, err := c.authFn(authPayload.Token)
	if err != nil {
		c.hub.log.Warn("WebSocket auth failed", "error", err)
		return false
	}

	c.userID = userID
	c.organizationID = orgID
	c.authenticated = true

	// Register with hub now that we're authenticated
	c.hub.Register(c)

	c.hub.log.Info("WebSocket client authenticated via message",
		"user_id", userID, "org_id", orgID)
	return true
}

// handleMessage processes incoming messages from the client
func (c *Client) handleMessage(data []byte) {
	var msg WSMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		c.hub.log.Error("Failed to unmarshal client message", "error", err)
		return
	}

	switch msg.Type {
	case TypeSetContact:
		c.handleSetContact(msg.Payload)
	case TypePing:
		c.sendPong()
	}
}

// handleSetContact updates the client's current contact
func (c *Client) handleSetContact(payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	var setContact SetContactPayload
	if err := json.Unmarshal(data, &setContact); err != nil {
		return
	}

	if setContact.ContactID == "" {
		c.currentContact = nil
		c.hub.log.Debug("Client cleared current contact", "user_id", c.userID)
	} else {
		contactID, err := uuid.Parse(setContact.ContactID)
		if err != nil {
			return
		}
		if c.contactAccessFn != nil && !c.contactAccessFn(c.userID, c.organizationID, contactID) {
			c.hub.log.Warn("Client attempted unauthorized contact subscription",
				"user_id", c.userID,
				"org_id", c.organizationID,
				"contact_id", contactID)
			return
		}
		c.currentContact = &contactID
		c.hub.log.Debug("Client set current contact",
			"user_id", c.userID,
			"contact_id", contactID)
	}
}

// SendChan returns the client's send channel for use in tests.
func (c *Client) SendChan() <-chan []byte {
	return c.send
}

// sendPong sends a pong response to the client
func (c *Client) sendPong() {
	msg := WSMessage{Type: TypePong}
	data, _ := json.Marshal(msg)
	select {
	case c.send <- data:
	default:
	}
}
