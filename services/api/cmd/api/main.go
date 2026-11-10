package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/paybridge/api/internal/claims"
	"github.com/paybridge/api/internal/config"
	"github.com/paybridge/api/internal/httpx"
	"github.com/paybridge/api/internal/orchestrator"
	"github.com/paybridge/api/internal/paypal"
	"github.com/paybridge/api/internal/reducer"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	httpClient := &http.Client{Timeout: 15 * time.Second}
	tm := paypal.NewTokenManager(cfg.PayPalClientID, cfg.PayPalClientSecret, cfg.PayPalAPIBase, httpClient)

	reader := paypal.NewReader(cfg.PayPalAPIBase, tm, httpClient)
	mutator := paypal.NewMutator(cfg.PayPalAPIBase, tm, httpClient)
	verifier := paypal.NewWebhookVerifier(cfg.PayPalAPIBase, tm, httpClient)
	red := reducer.NewReducer()
	val := claims.NewValidator()
	orc := orchestrator.NewOrchestrator(orchestrator.Config{
		LLMEnabled:     cfg.LLMEnabled,
		TimeoutSeconds: 30,
	}, val)

	srv := httpx.NewServer(cfg, red, reader, mutator, verifier, val, orc)

	httpServer := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      srv.Router(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("PayBridge AI HTTP Server listening on port %s (env: %s, paypal: %s)", cfg.Port, cfg.AppEnv, cfg.PayPalEnv)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down PayBridge AI HTTP Server gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	fmt.Println("PayBridge AI HTTP Server stopped gracefully.")
}
