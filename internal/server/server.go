package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type Params struct {
	Port int
}

type Server struct {
	p Params

	srv *http.Server
}

func NewServer(p Params) *Server {
	// Standard HTTP Mux server, no need for anything fancy
	mux := http.NewServeMux()
	// mux.Handle("/trades", tradesHandler)
	// mux.Handle("/ohlc", ohlcHandler)

	return &Server{
		p: p,
		srv: &http.Server{
			Addr:    fmt.Sprintf(":%d", p.Port),
			Handler: middleware(mux),
		},
	}
}

func (s *Server) Run(ctx context.Context) error {
	// Buffer the error channel so that the routine
	// pushing to it can exit immediately.
	errCh := make(chan error, 1)

	// Start serving.
	go func() {
		if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err // Abnormal termination event so push the error
			return
		}
		errCh <- nil // http.ErrServerClosed is a normal shutdown event
	}()

	// Wait for the context to end
	select {
	case <-ctx.Done():
		// Attempt a graceful shutdown with a timeout
		shCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_ = s.srv.Shutdown(shCtx) // We wll drop the error here since it's inconsequential
		<-errCh
		return nil

	case err := <-errCh:
		// Non-graceful server error (bind failure, etc.)
		return err
	}
}

// Simple request logging and validation middleware.
//
// This would be available as built in by some frameworks
// such as GIN.
func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			fmt.Printf("NON-GET REQUEST REJECTED: %s %s\n", r.Method, r.URL.Path)
			return
		}

		start := time.Now()
		next.ServeHTTP(w, r)
		fmt.Printf("%s %s %s\n", r.Method, r.URL.Path, time.Since(start))
	})
}
