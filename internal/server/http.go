package server

import (
	"log"
	"net"
	"net/http"
	"strconv"
	"time"
)

const readHeaderTimeout = 5 * time.Second

func New(addr string, store *Store) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/update-ip", store.HandleUpdatePhoneIP)

	return &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}
}

func (s *Store) HandleUpdatePhoneIP(w http.ResponseWriter, r *http.Request) {
	phoneIDStr := r.URL.Query().Get("id")
	if phoneIDStr == "" {
		log.Println("id not set in request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	phoneID, err := strconv.Atoi(phoneIDStr)
	if err != nil {
		log.Printf("error convert id (%s) from str to int\nmaybe you set incorrect id on phone", phoneIDStr)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	s.UpsertPhone(phoneID, ip)

	w.WriteHeader(http.StatusCreated)
}
