package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/nschatz/tracker/server/internal/api"
	"github.com/nschatz/tracker/server/internal/auth"
	"github.com/nschatz/tracker/server/internal/geo"
	"github.com/nschatz/tracker/server/internal/notify"
	"github.com/nschatz/tracker/server/internal/store"
	"github.com/nschatz/tracker/server/internal/ws"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	port := envOrDefault("PORT", "8080")
	dbURL := requireEnv("DATABASE_URL")
	jwtSecret := requireEnv("JWT_SECRET")
	ntfyURL := envOrDefault("NTFY_URL", "http://ntfy:80")
	retentionDays := envIntOrDefault("LOCATION_RETENTION_DAYS", 30)

	db, err := store.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	a := auth.New(jwtSecret)
	hub := ws.NewHub()
	go hub.Run()

	var sender notify.Sender
	if ntfyURL != "" {
		s, err := notify.NewNtfySender(ntfyURL)
		if err != nil {
			log.Fatalf("ntfy: %v", err)
		}
		sender = s
		log.Printf("ntfy sender configured: %s", ntfyURL)
	} else {
		log.Println("WARNING: NTFY_URL not set, using noop sender")
		sender = notify.NoopSender{}
	}
	notifier := notify.NewNotifier(sender)
	geoTracker := geo.NewTracker()

	srv := api.NewServer(a, db, db, db, db, hub, geoTracker, notifier, db)

	go runRetention(ctx, db, retentionDays)

	httpSrv := &http.Server{Addr: ":" + port, Handler: srv}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		httpSrv.Shutdown(shutdownCtx)
	}()

	log.Printf("listening on :%s", port)
	if err := httpSrv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("http: %v", err)
	}
}

func runRetention(ctx context.Context, db interface {
	DeleteLocationsOlderThan(context.Context, int) (int64, error)
}, days int) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			count, err := db.DeleteLocationsOlderThan(ctx, days)
			if err != nil {
				log.Printf("retention: %v", err)
			} else if count > 0 {
				log.Printf("retention: deleted %d old location rows", count)
			}
		}
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}

func envIntOrDefault(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		log.Fatalf("env var %s must be an integer: %v", key, err)
	}
	return n
}
