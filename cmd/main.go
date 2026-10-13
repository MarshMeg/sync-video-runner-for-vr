package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/MarshMeg/sync-video-runner-for-vr/internal/server"
	"github.com/MarshMeg/sync-video-runner-for-vr/internal/tui"
)

const (
	addr            = ":50050"
	shutdownTimeout = 3 * time.Second
)

func main() {
	store := server.NewStore()
	srv := server.New(addr, store)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen on %s: %v", addr, err)
	}
	fmt.Printf("Server started on %s\n", addr)

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.Serve(ln)
	}()

	tui.Run(store)

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("server shutdown: %v", err)
	}
	if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) {
		log.Printf("server stopped: %v", err)
	}
}
