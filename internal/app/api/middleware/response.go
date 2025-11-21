package middleware

import ("net/http"

	"github.com/go-chi/chi/v5/middleware")

// WrapResponseWriter wraps response writer to capture status
func NewWrapResponseWriter(w http.ResponseWriter, protoMajor int) middleware.WrapResponseWriter {
	return middleware.NewWrapResponseWriter(w, protoMajor)
}