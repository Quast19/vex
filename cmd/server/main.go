package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Quast19/vex/internal/engine"
	"github.com/Quast19/vex/internal/ordergateway"
)

func main() {
	// 1. Cancel root context automatically on Ctrl+C (SIGINT) or SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 2. Initialize and start the single-writer matching engine
	eng := engine.NewEngine(4096, 4096)
	eng.Start(ctx)

	// 3. Wire the HTTP order gateway
	srv := ordergateway.NewServer(eng)
	httpServer := &http.Server{
		Addr:         ":8080",
		Handler:      srv.Routes(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	// 4. Run the HTTP server in the background
	go func() {
		log.Println("PulseBook server listening on :8080")
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server error: %v", err)
		}
	}()

	// 5. Block until shutdown signal, then drain in-flight HTTP requests
	<-ctx.Done()
	log.Println("Shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown error: %v", err)
	}
}
