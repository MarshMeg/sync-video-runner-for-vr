package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type StartResult struct {
	Phone Phone
	Err   error
}

type Starter struct {
	client *http.Client
}

func NewStarter(timeout time.Duration) *Starter {
	return &Starter{
		client: &http.Client{Timeout: timeout},
	}
}

// Start sends GET /start to every phone concurrently and waits for all of
// them. Results are returned in the order of phones.
func (s *Starter) Start(ctx context.Context, phones []Phone) []StartResult {
	results := make([]StartResult, len(phones))

	var wg sync.WaitGroup
	for i, phone := range phones {
		wg.Add(1)

		go func() {
			defer wg.Done()
			results[i] = StartResult{Phone: phone, Err: s.start(ctx, phone)}
		}()
	}
	wg.Wait()

	return results
}

func (s *Starter) start(ctx context.Context, phone Phone) error {
	addr := net.JoinHostPort(phone.IP, strconv.Itoa(int(phone.Port)))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+addr+"/start", nil)
	if err != nil {
		return err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}

	return nil
}
