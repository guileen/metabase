package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

// CORS middleware
func CORS(next http.Handler) http.Handler {
	return middleware.AllowContentType("application/json", "text/plain")(next)
}
