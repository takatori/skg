package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/takatori/skg/internal"
	"github.com/takatori/skg/internal/server"
)

func main() {

	config, err := internal.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load config: ", err)
	}

	logger := internal.NewLogger(config)
	slog.SetDefault(logger)

	e, err := server.InitServer(config)
	if err != nil {
		log.Fatal("Failed to initialize server: ", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		if err := e.Start(config.EchoAddr); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("shutting down the server")
		}
	}()

	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}
