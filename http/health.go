package http

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/snabble/go-logging"
)

type HealthCheckFunc func() error
type HealthPermit func(r *http.Request) bool

type result struct {
	Status string `json:"status"`
}

var (
	statusUp   = result{Status: "UP"}
	statusDown = result{Status: "DOWN"}
)

func HealthRoute(
	root *mux.Router,
	healthCheckFunc HealthCheckFunc,
	permit HealthPermit,
) {
	root.Handle("/health", checkHealthAccessHandler(permit, createHealthHandler(healthCheckFunc))).
		Methods(http.MethodGet)
}

func createHealthHandler(healthCheckFunc HealthCheckFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := healthCheckFunc()
		if err == nil {
			sendHealthCheckResult(200, statusUp, w)
		} else {
			logging.Application(r.Header).WithError(err).Error("healthcheck returned an error")
			sendHealthCheckResult(500, statusDown, w)
		}
	})
}

// checkHealthAccessHandler returns a new handler, which checks, that
// that the client is permitted to access the health.
func checkHealthAccessHandler(permit HealthPermit, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !permit(r) {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func sendHealthCheckResult(statusCode int, result result, w http.ResponseWriter) {
	out, _ := json.Marshal(result)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(out)
}
