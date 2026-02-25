package main

import (
	"context"
	"log"
	"os"
	"time"

	"boxpilot/server/internal/api"
	"boxpilot/server/internal/service"
	"boxpilot/server/internal/store"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "/data/app.db"
	}
	db, err := store.Open(dbPath)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer db.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go service.StartSubscriptionScheduler(ctx, db.DB, 30*time.Second)

	addr := ":8080"
	if a := os.Getenv("ADDR"); a != "" {
		addr = a
	}
	r := api.Router(db.DB)
	log.Printf("listen %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
