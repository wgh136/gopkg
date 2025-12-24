package main

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	serverstatus "github.com/wgh136/gopkg/server_status"
)

func main() {
	// Register the server status collector with the default Prometheus registry
	if err := serverstatus.Register(nil, "server1"); err != nil {
		log.Fatalf("Failed to register server status collector: %v", err)
	}

	log.Println("Server status metrics registered successfully")
	log.Println("Starting HTTP server on :8080")
	log.Println("Metrics available at http://localhost:8080/metrics")

	// Expose the /metrics endpoint
	http.Handle("/metrics", promhttp.Handler())

	// Start the HTTP server
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}
