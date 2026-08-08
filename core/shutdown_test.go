package core

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func listenEphemeral(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	return ln
}

func TestServe_ReturnsNilOnGracefulShutdown(t *testing.T) {
	gadget := Gadget{}
	ln := listenEphemeral(t)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- gadget.serve(ctx, ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
	}()

	// Give the server a moment to start accepting connections.
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("serve did not return after context cancellation")
	}
}

func TestServe_DrainsInFlightRequestBeforeReturning(t *testing.T) {
	gadget := Gadget{shutdownTimeout: 2 * time.Second}
	ln := listenEphemeral(t)
	addr := ln.Addr().String()
	ctx, cancel := context.WithCancel(context.Background())

	handlerStarted := make(chan struct{})
	releaseHandler := make(chan struct{})
	handlerCompleted := make(chan struct{})

	done := make(chan error, 1)
	go func() {
		done <- gadget.serve(ctx, ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			close(handlerStarted)
			<-releaseHandler
			w.WriteHeader(http.StatusOK)
			close(handlerCompleted)
		}))
	}()

	// Fire a request that blocks inside the handler until we release it.
	reqDone := make(chan struct{})
	go func() {
		resp, err := http.Get("http://" + addr) //nolint:noctx // test helper
		if err == nil {
			_ = resp.Body.Close()
		}
		close(reqDone)
	}()

	select {
	case <-handlerStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("handler never started")
	}

	// Cancel while the request is still in flight, then release it shortly after.
	cancel()
	time.Sleep(50 * time.Millisecond)
	select {
	case <-handlerCompleted:
		t.Fatal("handler completed before it was released")
	default:
	}
	close(releaseHandler)

	select {
	case err := <-done:
		assert.NoError(t, err, "graceful shutdown should wait for the in-flight request to drain")
	case <-time.After(2 * time.Second):
		t.Fatal("serve did not return after in-flight request drained")
	}

	<-reqDone
}

func TestServe_ReturnsErrorWhenShutdownTimesOutWithSlowRequest(t *testing.T) {
	gadget := Gadget{shutdownTimeout: 50 * time.Millisecond}
	ln := listenEphemeral(t)
	addr := ln.Addr().String()
	ctx, cancel := context.WithCancel(context.Background())

	handlerStarted := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- gadget.serve(ctx, ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			close(handlerStarted)
			time.Sleep(1 * time.Second) // longer than shutdownTimeout
			w.WriteHeader(http.StatusOK)
		}))
	}()

	go func() {
		resp, err := http.Get("http://" + addr) //nolint:noctx // test helper
		if err == nil {
			_ = resp.Body.Close()
		}
	}()

	select {
	case <-handlerStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("handler never started")
	}

	cancel()

	select {
	case err := <-done:
		assert.Error(t, err, "expected shutdown to time out while the slow request was still in flight")
	case <-time.After(2 * time.Second):
		t.Fatal("serve did not return after shutdown timeout should have elapsed")
	}
}

func TestServe_DefaultsShutdownTimeoutWhenUnset(t *testing.T) {
	gadget := Gadget{} // shutdownTimeout left at zero value
	ln := listenEphemeral(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled; serve should shut down immediately

	err := gadget.serve(ctx, ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	assert.NoError(t, err)
}
