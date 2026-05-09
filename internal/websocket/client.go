package websocket

import (
	"encoding/json"
	"sync"
	"sync/atomic"
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

)

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

	// Function to validate contact subscriptions.
	contactAccessFn ContactAccessFn

	// Current contact being viewed (nil if none)
	currentContactMu sync.RWMutex
	currentContact   *uuid.UUID

	// dropped counts messages dropped because the per-client send buffer was full
	dropped atomic.Int64
}

// NewClient creates a new pre-authenticated Client instance.
func NewClient(hub *Hub, conn *websocket.Conn, userID, orgID uuid.UUID) *Client {
	return &Client{
		hub:            hub,
		conn:           conn,
		send:           make(chan []byte, clientSendBufSize),
		userID:         userID,
		organizationID: orgID,
		authenticated:  userID != uuid.Nil, // pre-authenticated if userID is set (tests)
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
		c.hub.unregister <- c
		c.connMu.Lock()
		if c.conn != nil {
			_ = c.conn.Close()
			c.conn = nil
		}
		c.connMu.Unlock()
	}()

	c.conn.SetReadLimit(maxMessageSize)
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
		c.setCurrentContact(nil)
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
		c.setCurrentContact(&contactID)
		c.hub.log.Debug("Client set current contact",
			"user_id", c.userID,
			"contact_id", contactID)
	}
}

func (c *Client) setCurrentContact(contactID *uuid.UUID) {
	c.currentContactMu.Lock()
	if contactID == nil {
		c.currentContact = nil
	} else {
		idCopy := *contactID
		c.currentContact = &idCopy
	}
	c.currentContactMu.Unlock()
}

func (c *Client) getCurrentContact() *uuid.UUID {
	c.currentContactMu.RLock()
	defer c.currentContactMu.RUnlock()
	if c.currentContact == nil {
		return nil
	}
	idCopy := *c.currentContact
	return &idCopy
}

// SendChan returns the client's send channel for use in tests.
func (c *Client) SendChan() <-chan []byte {
	return c.send
}

// DroppedCount returns the number of messages dropped for this client.
func (c *Client) DroppedCount() int64 {
	return c.dropped.Load()
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
