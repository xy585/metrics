package main

import (
	"fmt"
	"net/http"

	"k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/legacyregistry"
)

func main() {
	// Initialize the metric
	StartedContainersErrorsTotal := metrics.NewCounterVec(
		&metrics.CounterOpts{
			Subsystem:      "kubelet",
			Name:           "started_containers_errors_total",
			Help:           "Cumulative number of errors when starting containers",
			StabilityLevel: "ALPHA",
		},
		[]string{"container_type", "code"},
	)

	// Register the metric with the global registry
	legacyregistry.MustRegister(StartedContainersErrorsTotal)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// userInput comes from the query parameter
		userInput := r.URL.Query().Get("userInput")
		if userInput == "" {
			userInput = "none"
		}

		StartedContainersErrorsTotal.WithLabelValues("sandbox", userInput).Inc()
		fmt.Fprintf(w, "Recorded metric for userInput: %s\n", userInput)
	})

	// Expose metrics endpoint to verify
	http.Handle("/metrics", legacyregistry.Handler())

	fmt.Println("Server listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
