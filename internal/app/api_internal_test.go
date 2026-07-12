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

func TestRunForceClosesServerAfterShutdownTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	serveStopped := make(chan struct{})
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
		<-shutdownCtx.Done()
		return shutdownCtx.Err()
	}
	closeCalls := 0
	application.closeServer = func() error {
		closeCalls++
		close(serveStopped)
		return nil
	}

	runDone := make(chan error, 1)
	go func() { runDone <- application.Run(ctx) }()
	cancel()

	select {
	case err := <-runDone:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run did not finish after forced close")
	}
	if shutdownCalls != 1 || closeCalls != 1 {
		t.Fatalf("shutdown calls = %d, close calls = %d", shutdownCalls, closeCalls)
	}
}

func TestCloseStopsServerBeforeReturning(t *testing.T) {
	application := &API{
		server:          &http.Server{},
		logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
		shutdownTimeout: time.Second,
	}
	shutdownCalled := false
	application.shutdown = func(context.Context) error {
		shutdownCalled = true
		return nil
	}
	application.Close()
	application.Close()
	if !shutdownCalled {
		t.Fatal("Close did not stop HTTP server")
	}
}

func TestRunStopsShutdownWatcherWhenServeFails(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	serveFailure := errors.New("listen failed")
	shutdownCalled := make(chan struct{}, 1)
	application := &API{
		server:          &http.Server{},
		logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
		shutdownTimeout: time.Second,
		serve:           func() error { return serveFailure },
		shutdown: func(context.Context) error {
			shutdownCalled <- struct{}{}
			return nil
		},
	}
	if err := application.Run(ctx); !errors.Is(err, serveFailure) {
		t.Fatalf("Run error = %v", err)
	}
	cancel()
	select {
	case <-shutdownCalled:
		t.Fatal("shutdown watcher survived serve failure")
	case <-time.After(20 * time.Millisecond):
	}
}
