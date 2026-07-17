package main

import (
	"log"

	"app-booking/internal/config"
	"app-booking/internal/db"
	"app-booking/internal/server"
)

func main() {
	cfg := config.Load()

	conn, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("db connection failed: %v", err)
	}
	defer conn.Close()

	srv := server.New(cfg, conn)
	if err := srv.Run(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
