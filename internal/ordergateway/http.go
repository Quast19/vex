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
	"github.com/Quast19/vex/internal/engine"
)

type Server struct {
	eng    *engine.Engine
	nextID atomic.Uint64
}

func NewServer(eng *engine.Engine) *Server {
	return &Server{eng: eng}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /orders", s.handleCreateOrder)
	mux.HandleFunc("DELETE /orders/{id}", s.handleCancelOrder)
	return withCORS(mux)
}

type createOrderRequest struct {
	Side  string `json:"side"`
	Price int64  `json:"price"`
	Qty   int64  `json:"qty"`
}

func (s *Server) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	var req createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	var side book.Side
	switch req.Side {
	case "buy", "BUY":
		side = book.SideBuy
	case "sell", "SELL":
		side = book.SideSell
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "side must be buy or sell"})
		return
	}

	if req.Price <= 0 || req.Qty <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "price and qty must be positive"})
		return
	}

	id := s.nextID.Add(1)
	ctx, cancel := context.WithTimeout(r.Context(), 50*time.Millisecond)
	defer cancel()

	trades, err := s.eng.SubmitLimit(ctx, id, side, req.Price, req.Qty, time.Now().UnixNano())
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, engine.ErrEngineStopped) {
			status = http.StatusServiceUnavailable
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"order_id": id,
		"trades":   trades,
	})
}

func (s *Server) handleCancelOrder(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil || id == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid order id"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 50*time.Millisecond)
	defer cancel()

	cancelled, err := s.eng.SubmitCancel(ctx, id)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, engine.ErrEngineStopped) {
			status = http.StatusServiceUnavailable
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	if !cancelled {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"order_id":  id,
		"cancelled": true,
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
