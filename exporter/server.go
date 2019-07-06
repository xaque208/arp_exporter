package exporter

import (
	"encoding/json"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

// HealthStatus represents the availability of the components for this exporter.
type HealthStatus struct{}

func statusHandler(w http.ResponseWriter, r *http.Request) {

	health := &HealthStatus{}

	bytes, err := json.MarshalIndent(health, "", "\t")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = w.Write(bytes)
	if err != nil {
		log.Error(err)
	}
}

// StartMetricsServer is spawned in a Go routine to listen on http for /metrics requests.
func StartMetricsServer(bindAddr string) {
	d := http.NewServeMux()
	d.Handle("/metrics", promhttp.Handler())
	d.HandleFunc("/status/check", statusHandler)

	err := http.ListenAndServe(bindAddr, d)
	if err != nil {
		log.Fatal("Failed to start metrics server, error is:", err)
	}
}
