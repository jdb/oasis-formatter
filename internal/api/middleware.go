package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
)

func generateRequestID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "0000000000000000"
	}
	return hex.EncodeToString(b[:])
}

// WrapMiddleware applies request-ID generation, origin-aware CORS headers, and security headers
// matching the mespousses.fr edge/origin contract.
func WrapMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-Id")
		if reqID == "" {
			reqID = generateRequestID()
		}
		w.Header().Set("X-Request-Id", reqID)
		w.Header().Add("Vary", "Origin")

		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

		if r.Method == http.MethodOptions {
			w.Header().Add("Vary", "Access-Control-Request-Method")
			w.Header().Add("Vary", "Access-Control-Request-Headers")
			w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS")
			if reqHeaders := r.Header.Get("Access-Control-Request-Headers"); reqHeaders != "" {
				w.Header().Set("Access-Control-Allow-Headers", reqHeaders)
			} else {
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			}
			w.Header().Set("Access-Control-Max-Age", "600")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ProblemDetail represents an RFC 9457 application/problem+json response matching mespousses.fr.
type ProblemDetail struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Instance string `json:"instance"`
	Code     string `json:"code"`
	Detail   string `json:"detail,omitempty"`
}

func writeProblemJSON(w http.ResponseWriter, status int, code, title, instance, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ProblemDetail{
		Type:     "/problems/" + strings.ReplaceAll(code, "_", "-"),
		Title:    title,
		Status:   status,
		Instance: instance,
		Code:     code,
		Detail:   detail,
	})
}
