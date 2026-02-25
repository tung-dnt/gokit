package config

import (
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		env       map[string]string
		wantHost  string
		wantPort  string
		wantRead  time.Duration
		wantWrite time.Duration
		wantIdle  time.Duration
	}{
		{
			name:      "all defaults",
			env:       map[string]string{},
			wantHost:  "0.0.0.0",
			wantPort:  "8080",
			wantRead:  10 * time.Second,
			wantWrite: 10 * time.Second,
			wantIdle:  60 * time.Second,
		},
		{
			name: "custom values",
			env: map[string]string{
				"SERVER_HOST":          "127.0.0.1",
				"SERVER_PORT":          "3000",
				"SERVER_READ_TIMEOUT":  "5s",
				"SERVER_WRITE_TIMEOUT": "15s",
				"SERVER_IDLE_TIMEOUT":  "120s",
			},
			wantHost:  "127.0.0.1",
			wantPort:  "3000",
			wantRead:  5 * time.Second,
			wantWrite: 15 * time.Second,
			wantIdle:  120 * time.Second,
		},
		{
			name:      "invalid duration falls back to default",
			env:       map[string]string{"SERVER_READ_TIMEOUT": "not-a-duration"},
			wantHost:  "0.0.0.0",
			wantPort:  "8080",
			wantRead:  10 * time.Second,
			wantWrite: 10 * time.Second,
			wantIdle:  60 * time.Second,
		},
		{
			name:      "valid duration string",
			env:       map[string]string{"SERVER_READ_TIMEOUT": "500ms"},
			wantHost:  "0.0.0.0",
			wantPort:  "8080",
			wantRead:  500 * time.Millisecond,
			wantWrite: 10 * time.Second,
			wantIdle:  60 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			getenv := func(key string) string { return tt.env[key] }
			cfg := Load(getenv)

			assertServerConfig(t, cfg.Server, tt.wantHost, tt.wantPort, tt.wantRead, tt.wantWrite, tt.wantIdle)
		})
	}
}

func assertServerConfig(t *testing.T, s ServerConfig, host, port string, read, write, idle time.Duration) {
	t.Helper()
	if s.Host != host {
		t.Errorf("Host = %q, want %q", s.Host, host)
	}
	if s.Port != port {
		t.Errorf("Port = %q, want %q", s.Port, port)
	}
	if s.ReadTimeout != read {
		t.Errorf("ReadTimeout = %v, want %v", s.ReadTimeout, read)
	}
	if s.WriteTimeout != write {
		t.Errorf("WriteTimeout = %v, want %v", s.WriteTimeout, write)
	}
	if s.IdleTimeout != idle {
		t.Errorf("IdleTimeout = %v, want %v", s.IdleTimeout, idle)
	}
}
