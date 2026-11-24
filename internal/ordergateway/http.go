package ordergateway

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/Quast19/vex/internal/book"
	"github.com/Quast19/vex/internal/broadcaster"
	"github.com/Quast19/vex/internal/engine"
)

type Server struct {
	eng    *engine.Engine
	hub    *broadcaster.Hub
	nextID atomic.Uint64
}

func NewServer(eng *engine.Engine, hub *broadcaster.Hub) *Server {
	return &Server{
		eng: eng,
		hub: hub,
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /ws", s.hub.ServeWS)
	mux.HandleFunc("POST /orders", s.handleCreateOrder)
	mux.HandleFunc("DELETE /orders/{id}", s.handleCancelOrder)
	return withCORS(mux)
}