package main

import (
	"context"
	"encrp/internal/app"
	"encrp/internal/logger"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	a, err := app.NewApp()
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		if err := recover(); err != nil {
			a.Stop()
			logger.Errf("Main()", "Fatal error stopping app: %v", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)
		<-sigChan
		cancel()
		a.Stop()
	}()

	err = a.Start(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
