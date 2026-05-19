package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"vaultfleet/internal/master/api"
	"vaultfleet/internal/master/backup"
	"vaultfleet/internal/master/db"
	"vaultfleet/internal/master/events"
	"vaultfleet/internal/master/notify"
	"vaultfleet/internal/master/ws"
)

func main() {
	dataDir := flag.String("data-dir", "/data", "path to master data directory")
	addr := flag.String("addr", ":8080", "HTTP listen address")
	flag.Parse()

	log.Printf("starting VaultFleet master data-dir=%s addr=%s", *dataDir, *addr)

	restored, err := backup.CheckAndRestore(*dataDir)
	if err != nil {
		log.Fatalf("backup restore check failed: %v", err)
	}
	if restored {
		log.Printf("restored data directory from backup.zip")
	}

	database, err := db.New(*dataDir)
	if err != nil {
		log.Fatalf("database initialization failed: %v", err)
	}

	hub := ws.NewHub()
	bus := events.NewBus()
	api.SubscribeAgentStateEvents(database, bus)
	notify.NewDispatcher(database, bus).Start()
	policyLookup := api.CurrentPolicyLookup(database)

	wsHandler := ws.NewHandler(
		hub,
		bus,
		api.AuthenticateAgentByToken(database),
		policyLookup,
		api.NewTaskResultProcessor(database),
	)
	wsHandler.PolicyAckProcessor = api.NewPolicyAckProcessor(database)
	wsHandler.AgentStateUpdater = api.NewAgentStateUpdater(database)
	bus.Subscribe(events.PolicyChanged, api.NewPolicyChangedPusher(database, hub, policyLookup).Handle)
	router := api.NewRouter(api.RouterConfig{
		Database:       database,
		Hub:            hub,
		EventBus:       bus,
		AgentWebSocket: wsHandler.HandleWebSocket,
	})

	server := &http.Server{
		Addr:              *addr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go ws.NewMonitor(hub, bus).Run(ctx)

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("VaultFleet master listening on %s", *addr)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		log.Printf("shutdown signal received")
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server failed: %v", err)
		}
		return
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown failed: %v", err)
	}

	if err := <-serverErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server failed during shutdown: %v", err)
	}
	log.Printf("VaultFleet master stopped")
}
