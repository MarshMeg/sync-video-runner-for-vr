package server

import (
	"sort"
	"sync"
)

const phonePort = 8080

type Phone struct {
	ID   int
	IP   string
	Port uint16
}

type Store struct {
	phones map[int]Phone
	mu     sync.Mutex
}

func NewStore() *Store {
	return &Store{
		phones: make(map[int]Phone),
	}
}

func (s *Store) UpsertPhone(id int, ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.phones[id] = Phone{
		ID:   id,
		IP:   ip,
		Port: phonePort,
	}
}

func (s *Store) SnapshotPhones() []Phone {
	s.mu.Lock()
	defer s.mu.Unlock()

	phones := make([]Phone, 0, len(s.phones))
	for _, phone := range s.phones {
		phones = append(phones, phone)
	}

	sort.Slice(phones, func(i, j int) bool {
		return phones[i].ID < phones[j].ID
	})

	return phones
}

func (s *Store) ClearPhones() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := len(s.phones)
	s.phones = make(map[int]Phone)

	return count
}
