package httpx

import "net/http"

// NotFound writes a 404 with a custom message.
func NotFound(w http.ResponseWriter, r *http.Request, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"error":"not_found","message":"` + message + `"}`))
}
