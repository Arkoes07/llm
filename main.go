// Command api is the Ikhtiar HTTP server. It runs schema migrations on startup (NOT the seed —
// production data is added via the API; seeding is opt-in for dev/testing), ensures the implicit
// player exists, then serves.
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

	"github.com/arkoes07/llm/internal/config"
	"github.com/arkoes07/llm/internal/handler"
	"github.com/arkoes07/llm/internal/service/grogapi"
)

const (
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 10 * time.Second
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	groqapiSvc := grogapi.New(&cfg)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler.New(groqapiSvc).Router(),
		ReadHeaderTimeout: readHeaderTimeout,
	}

	go func() {
		log.Println("llm api listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Println("server error", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	log.Println("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Println("shutdown error", "err", err)
	}
}
