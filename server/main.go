package main

import (
	"log"
	"os"

	"boxpilot/server/internal/api"
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
