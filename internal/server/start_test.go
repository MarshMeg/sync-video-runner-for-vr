package server

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func phoneFor(t *testing.T, id int, srv *httptest.Server) Phone {
	t.Helper()

	host, portStr, err := net.SplitHostPort(srv.Listener.Addr().String())
	if err != nil {
		t.Fatalf("split %q: %v", srv.Listener.Addr(), err)
	}

	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		t.Fatalf("parse port %q: %v", portStr, err)
	}

	return Phone{ID: id, IP: host, Port: uint16(port)}
}

func TestStarterStart(t *testing.T) {
	var hits atomic.Int32

	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/start" {
			t.Errorf("got %s %s, want GET /start", r.Method, r.URL.Path)
		}
		hits.Add(1)
	}))
	defer ok.Close()

	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failing.Close()

	down := httptest.NewServer(http.NotFoundHandler())
	unreachable := phoneFor(t, 3, down)
	down.Close()

	phones := []Phone{phoneFor(t, 1, ok), phoneFor(t, 2, failing), unreachable}
	results := NewStarter(time.Second).Start(context.Background(), phones)

	if len(results) != len(phones) {
		t.Fatalf("got %d results, want %d", len(results), len(phones))
	}
	for i, result := range results {
		if result.Phone != phones[i] {
			t.Errorf("result %d is for %+v, want %+v", i, result.Phone, phones[i])
		}
	}

	if results[0].Err != nil {
		t.Errorf("reachable device: unexpected error %v", results[0].Err)
	}
	if got := hits.Load(); got != 1 {
		t.Errorf("reachable device got %d requests, want 1", got)
	}
	if err := results[1].Err; err == nil || !strings.Contains(err.Error(), "status 500") {
		t.Errorf("failing device: got error %v, want status 500", err)
	}
	if results[2].Err == nil {
		t.Error("unreachable device: expected an error")
	}
}

func TestStarterStartNoPhones(t *testing.T) {
	if results := NewStarter(time.Second).Start(context.Background(), nil); len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}

// Every handler blocks until all devices have received the request, so the
// test can only pass if the requests are in flight at the same time.
func TestStarterStartIsConcurrent(t *testing.T) {
	const devices = 5

	var arrived sync.WaitGroup
	arrived.Add(devices)
	allArrived := make(chan struct{})
	go func() {
		arrived.Wait()
		close(allArrived)
	}()

	phones := make([]Phone, 0, devices)
	for id := 1; id <= devices; id++ {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			arrived.Done()
			select {
			case <-allArrived:
			case <-r.Context().Done():
			}
		}))
		defer srv.Close()

		phones = append(phones, phoneFor(t, id, srv))
	}

	for _, result := range NewStarter(2*time.Second).Start(context.Background(), phones) {
		if result.Err != nil {
			t.Errorf("device %d: %v", result.Phone.ID, result.Err)
		}
	}
}

func TestStarterStartTimeout(t *testing.T) {
	release := make(chan struct{})
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
	}))
	defer slow.Close()
	defer close(release)

	began := time.Now()
	results := NewStarter(50*time.Millisecond).Start(context.Background(), []Phone{phoneFor(t, 1, slow)})

	if results[0].Err == nil {
		t.Fatal("expected a timeout error")
	}
	if elapsed := time.Since(began); elapsed > time.Second {
		t.Errorf("Start took %v, want it bounded by the client timeout", elapsed)
	}
}
