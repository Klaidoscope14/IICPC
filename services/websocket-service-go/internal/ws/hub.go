package ws

import (
	"log/slog"
	"sync"
)

// Hub maintains the set of active clients and broadcasts messages to the clients.
type Hub struct {
	// Registered clients, grouped by benchmark ID.
	// Empty string ("") can be used for global subscriptions like global leaderboard.
	rooms map[string]map[*Client]bool

	// Inbound messages from the server to broadcast to a specific room.
	broadcast chan BroadcastMessage

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	mu sync.RWMutex

	logger *slog.Logger
}

type BroadcastMessage struct {
	Room    string // The benchmark ID, or "" for global
	Payload []byte // JSON payload to send
}

func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		broadcast:  make(chan BroadcastMessage, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		rooms:      make(map[string]map[*Client]bool),
		logger:     logger,
	}
}

// Register adds a new client to the hub.
func (h *Hub) Register(c *Client) {
	h.register <- c
}

// Unregister removes a client from the hub.
func (h *Hub) Unregister(c *Client) {
	h.unregister <- c
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if _, ok := h.rooms[client.Room]; !ok {
				h.rooms[client.Room] = make(map[*Client]bool)
			}
			h.rooms[client.Room][client] = true
			h.mu.Unlock()
			h.logger.Debug("Client registered", "room", client.Room, "client_id", client.ID)

		case client := <-h.unregister:
			h.mu.Lock()
			if connections, ok := h.rooms[client.Room]; ok {
				if _, ok := connections[client]; ok {
					delete(connections, client)
					close(client.send)
					if len(connections) == 0 {
						delete(h.rooms, client.Room)
					}
					h.logger.Debug("Client unregistered", "room", client.Room, "client_id", client.ID)
				}
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			connections := h.rooms[message.Room]
			for client := range connections {
				select {
				case client.send <- message.Payload:
				default:
					// If the client's send buffer is full, we assume the client is dead or stuck
					// and we drop the client.
					h.mu.RUnlock() // unlock to allow unregister
					h.unregister <- client
					h.mu.RLock() // re-lock to continue loop
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends a message to all clients in a specific room.
func (h *Hub) Broadcast(room string, payload []byte) {
	h.broadcast <- BroadcastMessage{
		Room:    room,
		Payload: payload,
	}
}
