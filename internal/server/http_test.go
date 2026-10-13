package server

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"sync"
	"testing"
)

func silenceLog(t *testing.T) {
	t.Helper()

	prev := log.Writer()
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(prev) })
}

func TestHandleUpdatePhoneIP(t *testing.T) {
	silenceLog(t)

	type request struct {
		target     string
		remoteAddr string
		wantStatus int
	}

	tests := []struct {
		name     string
		requests []request
		want     []Phone
	}{
		{
			name:     "registers a device",
			requests: []request{{"/update-ip?id=1", "192.168.1.10:54321", http.StatusCreated}},
			want:     []Phone{{ID: 1, IP: "192.168.1.10", Port: 8080}},
		},
		{
			name:     "missing id",
			requests: []request{{"/update-ip", "192.168.1.10:54321", http.StatusBadRequest}},
			want:     []Phone{},
		},
		{
			name:     "empty id",
			requests: []request{{"/update-ip?id=", "192.168.1.10:54321", http.StatusBadRequest}},
			want:     []Phone{},
		},
		{
			name:     "non-numeric id",
			requests: []request{{"/update-ip?id=abc", "192.168.1.10:54321", http.StatusBadRequest}},
			want:     []Phone{},
		},
		{
			name:     "fractional id",
			requests: []request{{"/update-ip?id=1.5", "192.168.1.10:54321", http.StatusBadRequest}},
			want:     []Phone{},
		},
		{
			name: "same id updates the address",
			requests: []request{
				{"/update-ip?id=1", "192.168.1.10:54321", http.StatusCreated},
				{"/update-ip?id=1", "192.168.1.20:40000", http.StatusCreated},
			},
			want: []Phone{{ID: 1, IP: "192.168.1.20", Port: 8080}},
		},
		{
			name: "several devices",
			requests: []request{
				{"/update-ip?id=2", "192.168.1.20:40000", http.StatusCreated},
				{"/update-ip?id=1", "192.168.1.10:54321", http.StatusCreated},
			},
			want: []Phone{
				{ID: 1, IP: "192.168.1.10", Port: 8080},
				{ID: 2, IP: "192.168.1.20", Port: 8080},
			},
		},
		{
			name:     "ipv6 remote address",
			requests: []request{{"/update-ip?id=1", "[fe80::1]:54321", http.StatusCreated}},
			want:     []Phone{{ID: 1, IP: "fe80::1", Port: 8080}},
		},
		{
			name:     "remote address without a port",
			requests: []request{{"/update-ip?id=1", "192.168.1.10", http.StatusInternalServerError}},
			want:     []Phone{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewStore()
			handler := New("", store).Handler

			for _, r := range tt.requests {
				req := httptest.NewRequest(http.MethodGet, r.target, nil)
				req.RemoteAddr = r.remoteAddr
				rec := httptest.NewRecorder()

				handler.ServeHTTP(rec, req)

				if rec.Code != r.wantStatus {
					t.Errorf("GET %s: status %d, want %d", r.target, rec.Code, r.wantStatus)
				}
			}

			if got := store.SnapshotPhones(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("store holds %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestServerRegistersRealClient(t *testing.T) {
	store := NewStore()
	srv := httptest.NewServer(New("", store).Handler)
	defer srv.Close()

	const devices = 20

	var wg sync.WaitGroup
	for id := 1; id <= devices; id++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			resp, err := srv.Client().Get(srv.URL + "/update-ip?id=" + strconv.Itoa(id))
			if err != nil {
				t.Errorf("device %d: %v", id, err)
				return
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusCreated {
				t.Errorf("device %d: status %d, want %d", id, resp.StatusCode, http.StatusCreated)
			}
		}()
	}
	wg.Wait()

	phones := store.SnapshotPhones()
	if len(phones) != devices {
		t.Fatalf("got %d devices, want %d", len(phones), devices)
	}
	for _, phone := range phones {
		if phone.IP != "127.0.0.1" {
			t.Errorf("device %d: IP %q, want 127.0.0.1", phone.ID, phone.IP)
		}
	}
}

func TestUnknownPathIsNotFound(t *testing.T) {
	rec := httptest.NewRecorder()
	New("", NewStore()).Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/start", nil))

	if rec.Code != http.StatusNotFound {
		t.Errorf("status %d, want %d", rec.Code, http.StatusNotFound)
	}
}
