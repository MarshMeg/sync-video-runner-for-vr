package server

import (
	"reflect"
	"sync"
	"testing"
)

func TestStoreSnapshotPhones(t *testing.T) {
	type upsert struct {
		id int
		ip string
	}

	tests := []struct {
		name    string
		upserts []upsert
		want    []Phone
	}{
		{
			name: "empty store",
			want: []Phone{},
		},
		{
			name:    "single device gets the default port",
			upserts: []upsert{{1, "10.0.0.1"}},
			want:    []Phone{{ID: 1, IP: "10.0.0.1", Port: 8080}},
		},
		{
			name:    "sorted by id regardless of insertion order",
			upserts: []upsert{{3, "10.0.0.3"}, {1, "10.0.0.1"}, {2, "10.0.0.2"}},
			want: []Phone{
				{ID: 1, IP: "10.0.0.1", Port: 8080},
				{ID: 2, IP: "10.0.0.2", Port: 8080},
				{ID: 3, IP: "10.0.0.3", Port: 8080},
			},
		},
		{
			name:    "same id overwrites the address",
			upserts: []upsert{{1, "10.0.0.1"}, {1, "10.0.0.9"}},
			want:    []Phone{{ID: 1, IP: "10.0.0.9", Port: 8080}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewStore()
			for _, u := range tt.upserts {
				store.UpsertPhone(u.id, u.ip)
			}

			if got := store.SnapshotPhones(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestStoreSnapshotIsACopy(t *testing.T) {
	store := NewStore()
	store.UpsertPhone(1, "10.0.0.1")

	snapshot := store.SnapshotPhones()
	snapshot[0].IP = "changed"

	if got := store.SnapshotPhones()[0].IP; got != "10.0.0.1" {
		t.Errorf("store was modified through a snapshot: IP = %q", got)
	}
}

func TestStoreClearPhones(t *testing.T) {
	tests := []struct {
		name   string
		phones int
	}{
		{"empty store", 0},
		{"three devices", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewStore()
			for id := 1; id <= tt.phones; id++ {
				store.UpsertPhone(id, "10.0.0.1")
			}

			if got := store.ClearPhones(); got != tt.phones {
				t.Errorf("ClearPhones() = %d, want %d", got, tt.phones)
			}
			if got := len(store.SnapshotPhones()); got != 0 {
				t.Errorf("%d devices left after clear", got)
			}
		})
	}
}

// Meaningful under -race: writers, readers and clears hit the store at once.
func TestStoreConcurrentAccess(t *testing.T) {
	const (
		writers = 8
		ids     = 50
	)

	store := NewStore()

	var wg sync.WaitGroup
	for w := 0; w < writers; w++ {
		wg.Add(3)

		go func() {
			defer wg.Done()
			for id := 0; id < ids; id++ {
				store.UpsertPhone(id, "10.0.0.1")
			}
		}()
		go func() {
			defer wg.Done()
			for i := 0; i < ids; i++ {
				for _, phone := range store.SnapshotPhones() {
					_ = phone.IP
				}
			}
		}()
		go func() {
			defer wg.Done()
			for i := 0; i < ids; i++ {
				store.ClearPhones()
			}
		}()
	}
	wg.Wait()

	store.ClearPhones()
	for id := 0; id < ids; id++ {
		store.UpsertPhone(id, "10.0.0.1")
	}
	if got := len(store.SnapshotPhones()); got != ids {
		t.Errorf("got %d devices, want %d", got, ids)
	}
}
