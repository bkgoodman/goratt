package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// HTTPServer handles incoming HTTP print requests from the backend (authbackend).
type HTTPServer struct {
	server  *http.Server
	printer *Printer
}

// StartHTTPServer starts the HTTP listener on the configured port.
func StartHTTPServer(port int, printer *Printer) *HTTPServer {
	addr := fmt.Sprintf(":%d", port)
	mux := http.NewServeMux()

	s := &HTTPServer{
		printer: printer,
	}

	// Health check endpoint for monitoring tools
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok"}`+"\n")
	})

	// Primary print handler (handles POST / or POST /print_label)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Only allow POST requests for printing
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed. Print requests must use POST.", http.StatusMethodNotAllowed)
			return
		}

		if err := r.ParseForm(); err != nil {
			log.Printf("HTTP form parse error from %s: %v", r.RemoteAddr, err)
			http.Error(w, "Bad Request: unable to parse form", http.StatusBadRequest)
			return
		}

		name := strings.TrimSpace(r.Form.Get("name"))
		numberStr := strings.TrimSpace(r.Form.Get("number"))

		// Strict validation: both name and number must be present and non-empty.
		// This prevents stray requests (health checks, crawlers, empty submissions)
		// from creating blank "{0 }" ArUco labels!
		if name == "" || numberStr == "" {
			log.Printf("Rejected invalid ArUco print request from %s: name=%q, number=%q", r.RemoteAddr, name, numberStr)
			http.Error(w, "Bad Request: both 'name' and 'number' form parameters are required", http.StatusBadRequest)
			return
		}

		number, err := strconv.Atoi(numberStr)
		if err != nil || number < 0 || number >= len(aruco) {
			log.Printf("Rejected ArUco print request from %s with invalid code %q: %v", r.RemoteAddr, numberStr, err)
			http.Error(w, fmt.Sprintf("Bad Request: 'number' must be an integer between 0 and %d", len(aruco)-1), http.StatusBadRequest)
			return
		}

		log.Printf("Received valid ArUco print request: code=%d, member=%q from %s", number, name, r.RemoteAddr)

		if err := printer.PrintAruco(number, name); err != nil {
			log.Printf("Error printing ArUco label: %v", err)
			http.Error(w, "Printer error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","member":%q,"number":%d}`+"\n", name, number)
	})

	s.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		log.Printf("Starting HTTP print service on %s", addr)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	return s
}

// Shutdown gracefully stops the HTTP server.
func (s *HTTPServer) Shutdown() {
	if s.server == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	s.server.Shutdown(ctx)
}
