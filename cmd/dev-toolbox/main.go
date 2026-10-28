package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	if len(os.Args) == 1 && runDesktop() {
		return
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	code := run(ctx, os.Args[1:])
	os.Exit(code)
}
