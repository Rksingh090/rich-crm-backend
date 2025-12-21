package handlers

import (
	"net/http"
)

// HealthCheck godoc
// @Summary      Health Check
// @Description  Check if the server is up
// @Tags         health
// @Produce      plain
// @Success      200  {string}  string  "OK"
// @Router       /health [get]
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
