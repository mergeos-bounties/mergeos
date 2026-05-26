package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"mergeos/backend/internal/core"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	cfg := core.LoadConfig()
	payments := core.NewPaymentManager(cfg)
	repos := core.NewRepoFactory(cfg)
	emailer := core.NewEmailSender(cfg)
	store, err := core.NewStore(cfg, payments, repos, emailer)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()
	store.StartSSLReviewJob(context.Background())
	server := core.NewServer(cfg, store, payments)

	httpServer := &http.Server{
		Addr:              ":" + port,
		Handler:           server.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("MergeOS API listening on http://localhost:%s", port)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
