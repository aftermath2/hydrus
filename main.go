package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/aftermath2/hydrus/cmd/root"
)

func main() {
	root, err := root.NewCmd()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGHUP, syscall.SIGTERM)

	go func() {
		<-interrupt
		cancel()
	}()

	if err := root.ExecuteContext(ctx); err != nil {
		log.Fatal(err)
	}
}
