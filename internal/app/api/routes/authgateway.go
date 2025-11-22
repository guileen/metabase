package routes

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/guileen/metabase/internal/app/api/handlers"
	appmiddleware "github.com/guileen/metabase/internal/app/api/middleware"
)

// AuthGatewayRoutes registers authentication gateway routes
func AuthGatewayRoutes(r chi.Router, authGatewayHandler *handlers.AuthGatewayHandler) {
	// Apply middleware for auth gateway routes
	r.Group(func(r chi.Router) {
		// Request logging and recovery
		r.Use(middleware.Logger)
		r.Use(middleware.Recoverer)
		r.Use(middleware.AllowContentType("application/json"))
		r.Use(middleware.Timeout(30)) // 30 second timeout

		// Health check endpoint (no authentication required)
		r.Get("/health", authGatewayHandler.Health)

		// Public authentication endpoints (no authentication required)
		r.Route("/auth", func(r chi.Router) {
			// Authentication endpoints
			r.Post("/authenticate", authGatewayHandler.Authenticate)
			r.Post("/register", authGatewayHandler.Register)
			r.Post("/validate", authGatewayHandler.ValidateToken)
			r.Post("/refresh", authGatewayHandler.RefreshToken)

			// Password management
			r.Post("/password/reset", authGatewayHandler.ResetPassword)
			r.Post("/password/reset/confirm", authGatewayHandler.ConfirmResetPassword)

			// OAuth endpoints
			r.Route("/oauth", func(r chi.Router) {
				r.Get("/providers", authGatewayHandler.GetProviders)
				r.Get("/{provider}/login", authGatewayHandler.OAuthLogin)
				r.Get("/{provider}/callback", authGatewayHandler.OAuthCallback)
			})

			// Authentication providers
			r.Get("/providers", authGatewayHandler.GetProviders)
		})

		// Protected endpoints (authentication required)
		r.Group(func(r chi.Router) {
			// Apply JWT authentication middleware
			r.Use(appmiddleware.JWTAuth)

			// User management
			r.Route("/auth/user", func(r chi.Router) {
				r.Get("/", authGatewayHandler.GetUserInfo)
				r.Put("/", authGatewayHandler.UpdateUserInfo)
			})

			// Password management (authenticated)
			r.Post("/auth/password/change", authGatewayHandler.ChangePassword)

			// Session management
			r.Post("/auth/logout", authGatewayHandler.Logout)

			// Admin endpoints (require admin role)
			r.Group(func(r chi.Router) {
				// Apply RBAC middleware for admin access
				r.Use(appmiddleware.RequireRole("admin"))

				// Statistics and monitoring
				r.Get("/auth/stats", authGatewayHandler.GetStats)

				// User management (admin only)
				r.Route("/admin/users", func(r chi.Router) {
					// TODO: Add admin user management endpoints
				})
			})
		})
	})
}

// RegisterAuthGatewayMiddleware registers authentication gateway middleware
func RegisterAuthGatewayMiddleware(r chi.Router) {
	// Global middleware for auth gateway
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.NoCache)
	r.Use(middleware.Heartbeat("/ping"))

	// CORS middleware (adjust as needed for your security requirements)
	r.Use(appmiddleware.CORSWithConfig(appmiddleware.CORSConfig{
		AllowedOrigins: []string{"*"}, // Restrict in production
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		MaxAge:         300,
	}))

	// Rate limiting middleware (adjust as needed)
	r.Use(appmiddleware.RateLimitConfig(appmiddleware.RateLimiterConfig{
		Limit:  100,              // requests per minute
		Burst:  20,               // burst size
		Window: 60 * time.Second, // window in seconds
	}))

	// Security headers
	r.Use(appmiddleware.SecureHeaders)
}

// RegisterAuthGatewayV1Routes registers v1 API routes for authentication gateway
func RegisterAuthGatewayV1Routes(r chi.Router, authGatewayHandler *handlers.AuthGatewayHandler) {
	r.Route("/v1", func(r chi.Router) {
		AuthGatewayRoutes(r, authGatewayHandler)
	})
}
