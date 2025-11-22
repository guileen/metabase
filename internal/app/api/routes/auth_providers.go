package routes

import (
	"database/sql"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/guileen/metabase/internal/app/api/handlers"
	"github.com/guileen/metabase/internal/biz/domain/authgateway"
	"github.com/guileen/metabase/pkg/infra/auth"
)

// SetupAuthProvidersRoutes sets up authentication provider management routes
func SetupAuthProvidersRoutes(
	router *mux.Router,
	db *sql.DB,
	authGateway *authgateway.AuthGatewayManager,
	authMgr *auth.Manager,
) {
	// Auth providers handlers
	authProvidersHandler := handlers.NewAuthProvidersHandler(db, authGateway)
	userProfilesHandler := handlers.NewUserProfilesHandler(db, authGateway, authMgr)

	// Authentication provider management routes (admin only)
	authProviders := router.PathPrefix("/api/v1/auth/providers").Subrouter()

	// Provider CRUD operations
	authProviders.HandleFunc("", authProvidersHandler.GetProviders).Methods("GET")
	authProviders.HandleFunc("", authProvidersHandler.CreateProvider).Methods("POST")
	authProviders.HandleFunc("/{id}", authProvidersHandler.GetProvider).Methods("GET")
	authProviders.HandleFunc("/{id}", authProvidersHandler.UpdateProvider).Methods("PUT")
	authProviders.HandleFunc("/{id}", authProvidersHandler.DeleteProvider).Methods("DELETE")

	// Provider utilities
	authProviders.HandleFunc("/test", authProvidersHandler.TestProvider).Methods("POST")
	authProviders.HandleFunc("/stats", authProvidersHandler.GetProviderStats).Methods("GET")

	// User profile management routes (authenticated users)
	userProfiles := router.PathPrefix("/api/v1/user").Subrouter()

	// Profile management
	userProfiles.HandleFunc("/profile", userProfilesHandler.GetProfile).Methods("GET")
	userProfiles.HandleFunc("/profile", userProfilesHandler.UpdateProfile).Methods("PUT")
	userProfiles.HandleFunc("/profile/password", userProfilesHandler.ChangePassword).Methods("POST")

	// Connected accounts (OAuth)
	userProfiles.HandleFunc("/connected-accounts", userProfilesHandler.GetConnectedAccounts).Methods("GET")
	userProfiles.HandleFunc("/connected-accounts/connect", userProfilesHandler.ConnectAccount).Methods("POST")
	userProfiles.HandleFunc("/connected-accounts/{account_id}/disconnect", userProfilesHandler.DisconnectAccount).Methods("DELETE")
	userProfiles.HandleFunc("/profile/sync", userProfilesHandler.SyncProfile).Methods("POST")

	// OAuth authentication routes (public)
	oauth := router.PathPrefix("/api/v1/auth/oauth").Subrouter()
	oauth.HandleFunc("/{provider}/login", handleOAuthLogin).Methods("GET")
	oauth.HandleFunc("/{provider}/callback", handleOAuthCallback).Methods("GET")
}

// handleOAuthLogin initiates OAuth login flow
func handleOAuthLogin(w http.ResponseWriter, r *http.Request) {
	// This would be implemented to start the OAuth flow
	// Redirect user to provider's authorization URL
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte(`{"error": "OAuth login not implemented"}`))
}

// handleOAuthCallback handles OAuth callback from provider
func handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	// This would be implemented to handle OAuth callback
	// Exchange authorization code for tokens and create user session
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte(`{"error": "OAuth callback not implemented"}`))
}
