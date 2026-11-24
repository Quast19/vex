package broadcaster

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/Quast19/vex/internal/engine"
	"github.com/coder/websocket"
)

const clientBufferSize = 64

type client struct {
	conn *websocket.Conn
	send chan []byte
}

type Hub struct {
	events     <-chan engine.MarketEvent
	clients    map[*client]struct{}
	register   chan *client
	unregister chan *client
}

func NewHub(events <-chan engine.MarketEvent) *Hub {
	return &Hub{
		events:     events,
		clients:    make(map[*client]struct{}),
		register:   make(chan *client),
		unregister: make(chan *client),
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			for c := range h.clients {
				close(c.send)
				delete(h.clients, c)
			}
			return

		case c := <-h.register:
			h.clients[c] = struct{}{}

		case c := <-h.unregister:
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}

		case ev, ok := <-h.events:
			if !ok {
				return
			}
			if len(h.clients) == 0 {
				continue 
			}

			payload, err := json.Marshal(ev)
			if err != nil {
				log.Printf("broadcaster marshal error: %v", err)
				continue
			}

			for c := range h.clients {
				select {
				case c.send <- payload:
				default:
					// Slow consumer: drop client so it never blocks the engine
					delete(h.clients, c)
					close(c.send)
				}
			}
		}
	}
}

// ServeWS upgrades an HTTP connection to WebSocket and streams live market events.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"}, // Allow SvelteKit dev server
	})
	if err != nil {
		log.Printf("websocket accept error: %v", err)
		return
	}

	c := &client{
		conn: conn,
		send: make(chan []byte, clientBufferSize),
	}

	select {
	case h.register <- c:
	case <-r.Context().Done():
		_ = conn.Close(websocket.StatusGoingAway, "shutdown")
		return
	}

		select {
		case h.unregister <- c:
		default:
		}
		_ = conn.Close(websocket.StatusNormalClosure, "")
	}()

	for msg := range c.send {
		writeCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		err := conn.Write(writeCtx, websocket.MessageText, msg)
		cancel()
		if err != nil {
			return
		}
	}
}