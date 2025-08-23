package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func serveHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestHSTS(t *testing.T) {
	tests := []struct {
		host       string
		expectHsts bool
	}{
		{
			host:       "test-hostname",
			expectHsts: false,
		},
		{
			host:       "test-hostname.prawn-universe.ts.net",
			expectHsts: true,
		},
	}
	for _, tt := range tests {
		name := "host:[" + tt.host + "]"
		t.Run(name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			if tt.host != "" {
				r.Header.Add("Host", tt.host)
			}
			w := httptest.NewRecorder()
			HSTS(serveHandler()).ServeHTTP(w, r)
			_, found := w.Header()["Strict-Transport-Security"]
			if found != tt.expectHsts {
				t.Errorf("HSTS expectation: domain %s want: %t got: %t", tt.host, tt.expectHsts, found)
			}
		})
	}
}

func TestNonHTTPRedirectWithQuery(t *testing.T) {
	h := nonHTTPSHandlerFromHostname("foobar.com")
	r := httptest.NewRequest("GET", "http://example.com/?query=bar", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusFound {
		t.Errorf("got %d; want %d", w.Code, http.StatusFound)
	}
	if w.Header().Get("Location") != "https://foobar.com/?query=bar" {
		t.Errorf("got %q; want %q", w.Header().Get("Location"), "https://foobar.com/?query=bar")
	}
}

func TestValidateConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		config  *ServerConfig
		wantErr bool
	}{
		{
			name: "valid configuration",
			config: &ServerConfig{
				TailscaleAuthKey:        "tskey-test",
				Hostname:                "test-hostname",
				TailscaleStateDirectory: "/tmp/tailscale",
			},
			wantErr: false,
		},
		{
			name: "empty tailscale auth key",
			config: &ServerConfig{
				TailscaleAuthKey:        "",
				Hostname:                "test-hostname",
				TailscaleStateDirectory: "/tmp/tailscale",
			},
			wantErr: true,
		},
		{
			name: "empty hostname",
			config: &ServerConfig{
				TailscaleAuthKey:        "tskey-test",
				Hostname:                "",
				TailscaleStateDirectory: "/tmp/tailscale",
			},
			wantErr: true,
		},
		{
			name: "hostname with space",
			config: &ServerConfig{
				TailscaleAuthKey:        "tskey-test",
				Hostname:                "test hostname",
				TailscaleStateDirectory: "/tmp/tailscale",
			},
			wantErr: true,
		},
		{
			name: "hostname with dot",
			config: &ServerConfig{
				TailscaleAuthKey:        "tskey-test",
				Hostname:                "test.hostname",
				TailscaleStateDirectory: "/tmp/tailscale",
			},
			wantErr: true,
		},
		{
			name: "hostname with slash",
			config: &ServerConfig{
				TailscaleAuthKey:        "tskey-test",
				Hostname:                "test/hostname",
				TailscaleStateDirectory: "/tmp/tailscale",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfiguration(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfiguration() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
