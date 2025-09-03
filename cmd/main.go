package main

import (
	"context"
	"flag"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/infinityCounter2/vh-trader/internal/server"
)

var port int

func init() {
	flag.IntVar(&port, "port", 9001, "The default port the server should run on")
}

func main() {
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	httpServer := server.NewServer(server.Params{
		Port: port,
	})

	fmt.Printf("Starting server on port :%d\n", port)

	if err := httpServer.Run(ctx); err != nil {
		fmt.Printf("Server Run Error: %s\n", err)
	}

	fmt.Println("Server shutdown gracefully")
}
