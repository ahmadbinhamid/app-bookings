package main

import (
	"context"
	"log"

	"app-booking/internal/config"
	"app-booking/internal/db"
	"app-booking/internal/server"

	// Embeds the full IANA tz database into the binary — the Alpine runtime
	// image in Dockerfile has no /usr/share/zoneinfo, so without this,
	// time.LoadLocation("Europe/London") (location.Service.SetTimezone)
	// would fail for every real zone name once deployed, even though it
	// works fine in local dev on macOS/most Linux distros that do have
	// system zoneinfo installed.
	_ "time/tzdata"
)

func main() {
	cfg := config.Load()

	conn, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("db connection failed: %v", err)
	}
	defer conn.Close()

	srv := server.New(cfg, conn)

	// Background FlowPOS sync — see internal/modules/sync/scheduler.go for
	// why this is a plain goroutine rather than an external job/queue.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go srv.SyncScheduler.Start(ctx)

	if err := srv.Run(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
