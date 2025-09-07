# privateserver

A server library utilising Tailscale network

## Example

To run the following example, create an Auth key via Tailscale admin console,
create a new Go module and add the following code to your `main.go` file.

Run the example server with the following command.

```sh
go run main.go --ts-authkey=tskey-auth-aaa-bbbb --hostname=tailnet-test-service
```

Note that the following code assumes directory `tailscale-state` exists and is
writable.

```go
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/alexhokl/privateserver/server"
)

var (
	tsAuthKey = flag.String("ts-authkey", "", "Tailscale auth key")
	hostname  = flag.String("hostname", "", "Tailscale hostname for this server")
)

func main() {
	flag.Parse()

	if *tsAuthKey == "" {
		log.Fatal("Please provide a Tailscale auth key via option --ts-authkey")
	}
	if *hostname == "" {
		log.Fatal("Please provide a Tailscale hostname via option --hostname")
	}

	serverConfig := &server.ServerConfig{
		TailscaleAuthKey:        *tsAuthKey,
		Hostname:                *hostname,
		TailscaleStateDirectory: "./tailscale-state",
	}

	srv, err := server.NewServer(serverConfig)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	listeners, nonHTTPSListener, nonHTTPSHandler, err := srv.Listen([]int{443})
	if err != nil {
		log.Fatalf("Failed to start listening: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		who, err := srv.GetCallerIndentity(r)
		if err != nil {
			http.Error(w, "Failed to get caller identity", http.StatusInternalServerError)
		}

		_, err = fmt.Fprintf(w, "<html><body><h1>Hello %s from %s, world!</h1>\n", who.UserProfile.DisplayName, who.Node.Name)
		if err != nil {
			log.Printf("failed to write response: %v", err)
		}
	}))

	g, _ := errgroup.WithContext(context.Background())

	g.Go(func() error {
		log.Printf("Starting non-HTTPS server on %s", nonHTTPSListener.Addr().String())
		server := &http.Server{
			Handler:      nonHTTPSHandler,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		}
		return server.Serve(nonHTTPSListener)
	})

	for _, listener := range listeners {
		g.Go(func() error {
			log.Printf("Starting HTTPS server on %s", listener.Addr().String())
			server := &http.Server{
				Handler:      mux,
				ReadTimeout:  10 * time.Second,
				WriteTimeout: 10 * time.Second,
			}
			return server.Serve(listener)
		})
	}

	log.Fatal(g.Wait())
}
```
