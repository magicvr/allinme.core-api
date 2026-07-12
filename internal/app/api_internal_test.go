package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"
)

func TestRunWaitsForShutdownCompletionAfterTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	serveStopped := make(chan struct{})
	activeRequestDone := make(chan struct{})
	application := &API{
		server:          &http.Server{},
		logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
		shutdownTimeout: time.Millisecond,
		serve: func() error {
			<-serveStopped
			return http.ErrServerClosed
		},
	}
	shutdownCalls := 0
	application.shutdown = func(shutdownCtx context.Context) error {
		shutdownCalls++
		if shutdownCalls == 1 {
			close(serveStopped)
			<-shutdownCtx.Done()
			return shutdownCtx.Err()
		}
		<-activeRequestDone
		return nil
	}

	runDone := make(chan error, 1)
	go func() { runDone <- application.Run(ctx) }()
	cancel()

	select {
	case err := <-runDone:
		t.Fatalf("Run returned before active request completed: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	close(activeRequestDone)
	select {
	case err := <-runDone:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run did not finish after shutdown completed")
	}
	if shutdownCalls != 2 {
		t.Fatalf("shutdown calls = %d, want 2", shutdownCalls)
	}
}
