package tui

import (
	"errors"
	"testing"

	"github.com/MarshMeg/sync-video-runner-for-vr/internal/server"
)

func TestStartSummary(t *testing.T) {
	first := server.Phone{ID: 1, IP: "10.0.0.1", Port: 8080}
	second := server.Phone{ID: 2, IP: "10.0.0.2", Port: 8080}

	tests := []struct {
		name    string
		results []server.StartResult
		want    string
	}{
		{
			name: "no devices",
			want: "Ctrl+R: no connected devices.",
		},
		{
			name:    "all ok",
			results: []server.StartResult{{Phone: first}, {Phone: second}},
			want:    "Ctrl+R: request sent to 2 devices.",
		},
		{
			name: "partial failure reports the first error",
			results: []server.StartResult{
				{Phone: first},
				{Phone: second, Err: errors.New("status 500")},
			},
			want: "Ctrl+R: OK 1/2, failed 1. First error: 02 (10.0.0.2): status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := startSummary(tt.results); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
