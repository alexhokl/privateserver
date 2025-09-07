// Package server package provides functionality to create a server instance
package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"tailscale.com/client/local"
	"tailscale.com/client/tailscale/apitype"
	"tailscale.com/tsnet"
	"tailscale.com/util/dnsname"
)

const (
	HTTPAddress = ":80"
	Protocol    = "tcp"
)

type Server struct {
	tsServer *tsnet.Server
	tsClient *local.Client
	fqdn     string
}

type ServerConfig struct {
	TailscaleAuthKey        string
	Hostname                string
	TailscaleStateDirectory string
}

// NewServer creates and initializes a new Server instance based on the provided
// configuration.
func NewServer(config *ServerConfig) (*Server, error) {
	if err := validateConfiguration(config); err != nil {
		return nil, err
	}

	srv := new(Server)
	srv.tsServer = &tsnet.Server{
		AuthKey:  config.TailscaleAuthKey,
		Hostname: config.Hostname,
		Dir:      config.TailscaleStateDirectory,
	}

	// creates client to talk to Tailscale API
	tsClient, err := srv.tsServer.LocalClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create local client to talk to tailscale API: %w", err)
	}
	srv.tsClient = tsClient

	// loop until the Tailscale node is fully up and running
out:
	for {
		upCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		status, err := srv.tsServer.Up(upCtx)
		if err == nil && status != nil {
			break out
		}
	}

	// talks to Tailscale API to retrieve status of this node in tailnet
	statusCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	status, err := tsClient.Status(statusCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tailscale status: %w", err)
	}
	srv.fqdn = strings.TrimSuffix(status.Self.DNSName, ".")
	log.Printf("this service will be available on [%s]", srv.fqdn)

	return srv, nil
}

// Listen starts listening on the specified ports and returns the TLS listeners.
// If port 443 is among the specified ports, it also sets up a non-TLS listener
// on port 80 that redirects all HTTP requests to HTTPS.
func (s *Server) Listen(httpsPorts []int) (listeners []net.Listener, nonHTTPSListener net.Listener, nonHTTPSHandler http.Handler, err error) {
	listeners = make([]net.Listener, 0, len(httpsPorts))

	for _, port := range httpsPorts {
		addr := fmt.Sprintf(":%d", port)
		listener, err := s.tsServer.ListenTLS(Protocol, addr)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to listen TLS at [%s]: %w", addr, err)
		}
		listeners = append(listeners, listener)

		if port == 443 {
			nonHTTPSHandler = nonHTTPSHandlerFromHostname(s.fqdn)
			nonHTTPSListener, err = s.tsServer.Listen(Protocol, HTTPAddress)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed to listen non-TLS at [%s]: %w", HTTPAddress, err)
			}
		}
	}
	return listeners, nonHTTPSListener, nonHTTPSHandler, nil
}

// Close shuts down the tailscale server.
func (s *Server) Close() error {
	if s.tsServer == nil {
		return fmt.Errorf("server is not initialized")
	}
	return s.tsServer.Close()
}

// GetCallerIndentity retrieves the identity of the caller from the Tailscale
// API
func (s *Server) GetCallerIndentity(r *http.Request) (*apitype.WhoIsResponse, error) {
	who, err := s.tsClient.WhoIs(r.Context(), r.RemoteAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to get caller identity from tailscale API: %w", err)
	}
	return who, nil
}

func (s *Server) FQDN() string {
	return s.fqdn
}

// nonHTTPSHandlerFromHostname returns the http.Handler for serving all
// plaintext HTTP requests. It redirects all requests to the HTTPs version of
// the same URL.
func nonHTTPSHandlerFromHostname(hostname string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := &url.URL{
			Scheme:   "https",
			Host:     hostname,
			Path:     r.URL.Path,
			RawQuery: r.URL.RawQuery,
		}
		http.Redirect(w, r, u.String(), http.StatusFound)
	})
}

// HSTS wraps the provided handler and sets Strict-Transport-Security header on
// responses. It inspects the Host header to ensure we do not specify HSTS
// response on non fully qualified domain name origins.
func HSTS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, found := r.Header["Host"]
		if found {
			host := host[0]
			fqdn, err := dnsname.ToFQDN(host)
			if err == nil {
				segCount := fqdn.NumLabels()
				if segCount > 1 {
					w.Header().Set("Strict-Transport-Security", "max-age=31536000")
				}
			}
		}
		h.ServeHTTP(w, r)
	})
}

// validateConfiguration checks if the provided configuration is valid.
func validateConfiguration(config *ServerConfig) error {
	if config.TailscaleAuthKey == "" {
		return fmt.Errorf("tailscale auth key cannot be empty")
	}

	if config.Hostname == "" {
		return fmt.Errorf("hostname cannot be empty")
	}
	if strings.ContainsAny(config.Hostname, " ./") {
		return fmt.Errorf("hostname cannot contain space, dot, or slash")
	}

	return nil
}
