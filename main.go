package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"orders_backend/app"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	if err := app.Run(ctx); err != nil {
		log.Printf("Orders API stopped: %v", err)
		os.Exit(1)
	}
}
