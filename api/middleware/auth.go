package middleware

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/msenjem/aegis/api/config"
	"github.com/msenjem/aegis/api/model"
)

// AuthUser represents the authenticated user stored in the request context.
type AuthUser struct {
	ID    string
	Email string
	Name  string
	Role  string // "admin" or "viewer"
}

type contextKey string

const authUserKey contextKey = "auth_user"

// UserFromContext retrieves the authenticated user from the request context.
// Returns nil if no user is set (public routes).
func UserFromContext(ctx context.Context) *AuthUser {
	u, _ := ctx.Value(authUserKey).(*AuthUser)
	return u
}

// RequireAuth returns a middleware that enforces authentication on a handler.
// When cfg.AuthEnabled is false (local dev), it auto-authenticates using
// cfg.DevUserEmail. When true (production), it extracts user identity from
// the configured auth provider's headers (GCP IAP or Azure AD Easy Auth).
func RequireAuth(db *sql.DB, cfg *config.Config) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var user *AuthUser
			var err error

			if !cfg.AuthEnabled {
				user, err = devUser(r.Context(), db, cfg.DevUserEmail)
			} else {
				user, err = extractUser(r.Context(), db, r, cfg.AuthProvider)
			}

			if err != nil {
				writeAuthError(w, err)
				return
			}

			ctx := context.WithValue(r.Context(), authUserKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
	}
}

// RequireRole wraps RequireAuth and additionally checks that the authenticated
// user has the specified role. "admin" role has access to everything.
func RequireRole(role string, db *sql.DB, cfg *config.Config) func(http.HandlerFunc) http.HandlerFunc {
	authWrap := RequireAuth(db, cfg)
	return func(next http.HandlerFunc) http.HandlerFunc {
		return authWrap(func(w http.ResponseWriter, r *http.Request) {
			user := UserFromContext(r.Context())
			if user.Role != role && user.Role != "admin" {
				authWriteJSON(w, http.StatusForbidden, map[string]string{
					"error": "insufficient permissions: requires " + role + " role",
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type authError struct {
	Status  int
	Message string
}

func (e *authError) Error() string { return e.Message }

// extractUser determines the authenticated user based on the configured provider.
// Supported providers: "iap" (GCP), "azure" (Azure AD Easy Auth), "auto" (try both).
func extractUser(ctx context.Context, db *sql.DB, r *http.Request, provider string) (*AuthUser, error) {
	switch provider {
	case "iap":
		return iapUser(ctx, db, r)
	case "azure":
		return azureUser(ctx, db, r)
	default: // "auto" — try IAP first, then Azure
		if r.Header.Get("X-Goog-Authenticated-User-Email") != "" {
			return iapUser(ctx, db, r)
		}
		if r.Header.Get("X-MS-CLIENT-PRINCIPAL-NAME") != "" {
			return azureUser(ctx, db, r)
		}
		return nil, &authError{http.StatusUnauthorized, "missing authentication header"}
	}
}

func devUser(ctx context.Context, db *sql.DB, email string) (*AuthUser, error) {
	u, err := model.GetAdminUserByEmail(ctx, db, email)
	if err != nil {
		log.Printf("auth: dev user %q not found in admin_users, using synthetic admin", email)
		return &AuthUser{
			ID:    "00000000-0000-0000-0000-000000000000",
			Email: email,
			Name:  "Dev User",
			Role:  "admin",
		}, nil
	}
	if !u.Enabled {
		return nil, &authError{http.StatusForbidden, "account is disabled"}
	}
	return &AuthUser{ID: u.ID, Email: u.Email, Name: u.Name, Role: u.Role}, nil
}

// iapUser extracts user identity from GCP Identity-Aware Proxy headers.
// IAP sets X-Goog-Authenticated-User-Email to "accounts.google.com:user@example.com".
func iapUser(ctx context.Context, db *sql.DB, r *http.Request) (*AuthUser, error) {
	raw := r.Header.Get("X-Goog-Authenticated-User-Email")
	if raw == "" {
		return nil, &authError{http.StatusUnauthorized, "missing GCP IAP authentication header"}
	}

	// Strip "accounts.google.com:" prefix
	email := raw
	if idx := strings.Index(raw, ":"); idx != -1 {
		email = raw[idx+1:]
	}
	email = strings.TrimSpace(strings.ToLower(email))

	if email == "" {
		return nil, &authError{http.StatusUnauthorized, "invalid GCP IAP authentication header"}
	}

	return lookupUser(ctx, db, email)
}

// azureUser extracts user identity from Azure AD Easy Auth headers.
// Azure App Service Easy Auth sets X-MS-CLIENT-PRINCIPAL-NAME to the user's email.
// Azure Application Gateway + AAD also supports this header pattern.
func azureUser(ctx context.Context, db *sql.DB, r *http.Request) (*AuthUser, error) {
	email := r.Header.Get("X-MS-CLIENT-PRINCIPAL-NAME")
	if email == "" {
		return nil, &authError{http.StatusUnauthorized, "missing Azure AD authentication header"}
	}

	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return nil, &authError{http.StatusUnauthorized, "invalid Azure AD authentication header"}
	}

	return lookupUser(ctx, db, email)
}

// lookupUser finds a user by email in admin_users and checks they are enabled.
func lookupUser(ctx context.Context, db *sql.DB, email string) (*AuthUser, error) {
	u, err := model.GetAdminUserByEmail(ctx, db, email)
	if err != nil {
		return nil, &authError{http.StatusForbidden, "user not registered: " + email}
	}
	if !u.Enabled {
		return nil, &authError{http.StatusForbidden, "account is disabled"}
	}
	return &AuthUser{ID: u.ID, Email: u.Email, Name: u.Name, Role: u.Role}, nil
}

func writeAuthError(w http.ResponseWriter, err error) {
	if ae, ok := err.(*authError); ok {
		authWriteJSON(w, ae.Status, map[string]string{"error": ae.Message})
		return
	}
	authWriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "authentication error"})
}

func authWriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
