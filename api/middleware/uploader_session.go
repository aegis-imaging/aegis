package middleware

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
)

// UploaderSessionCookieName is the HttpOnly cookie name carrying the uploader
// session id. Distinct from any admin auth cookie so the two scopes never mix.
const UploaderSessionCookieName = "aegis_uploader_session"

// UploaderSessionTTL bounds how long a single login lasts before re-auth is
// required. Twelve hours is short enough that a stolen cookie isn't usable
// indefinitely, long enough that a coordinator can finish a multi-study
// upload session without re-typing the password.
const UploaderSessionTTL = 12 * time.Hour

// uploaderUserKey is the context key under which we stash the resolved AuthUser
// for an uploader-authenticated request. We reuse the existing AuthUser shape
// so handlers can call UserFromContext uniformly.
type uploaderUserKey struct{}

// RequireUploaderSession enforces a valid uploader session cookie on a handler.
// On success it populates the request context with the AuthUser (Role="uploader")
// just like RequireAuth does for admin users — so downstream handlers can call
// middleware.UserFromContext unchanged.
//
// Failure modes (all return 401 with a small JSON body):
//   - cookie missing
//   - cookie value doesn't match a row in uploader_sessions
//   - row exists but has expired
//   - row exists but the admin_users row is missing, disabled, or wrong role
func RequireUploaderSession(db *sql.DB) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user, sess, err := resolveUploaderSession(r.Context(), db, r)
			if err != nil {
				writeUploaderAuthError(w, err)
				return
			}
			// Best-effort touch — we don't block the request if it fails.
			sessID := sess.ID
			go func() {
				_ = model.TouchUploaderSession(context.Background(), db, sessID)
			}()
			ctx := context.WithValue(r.Context(), authUserKey, user)
			ctx = context.WithValue(ctx, uploaderUserKey{}, sess)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
	}
}

// UploaderSessionFromContext returns the resolved uploader session, or nil if
// the request was not authenticated as an uploader.
func UploaderSessionFromContext(ctx context.Context) *model.UploaderSession {
	s, _ := ctx.Value(uploaderUserKey{}).(*model.UploaderSession)
	return s
}

// resolveUploaderSession does the cookie -> session -> user chain. Extracted
// so login/me handlers can reuse the same checks via a public helper.
func resolveUploaderSession(ctx context.Context, db *sql.DB, r *http.Request) (*AuthUser, *model.UploaderSession, error) {
	cookie, err := r.Cookie(UploaderSessionCookieName)
	if err != nil || cookie.Value == "" {
		return nil, nil, &authError{http.StatusUnauthorized, "uploader session required"}
	}
	sess, err := model.GetUploaderSession(ctx, db, cookie.Value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, &authError{http.StatusUnauthorized, "session invalid or expired"}
		}
		return nil, nil, &authError{http.StatusInternalServerError, "session lookup failed"}
	}
	u, err := model.GetAdminUserByID(ctx, db, sess.UserID)
	if err != nil {
		return nil, nil, &authError{http.StatusUnauthorized, "user not found"}
	}
	if !u.Enabled {
		return nil, nil, &authError{http.StatusForbidden, "account is disabled"}
	}
	if u.Role != "uploader" && u.Role != "admin" {
		// Belt-and-suspenders — uploader sessions should only ever exist for
		// uploader-role users. If we somehow loaded one for a different role,
		// reject rather than silently grant upload rights.
		return nil, nil, &authError{http.StatusForbidden, "wrong account type for upload portal"}
	}
	return &AuthUser{ID: u.ID, Email: u.Email, Name: u.Name, Role: u.Role}, sess, nil
}

// ResolveUploaderSession is the exported variant used by login/me handlers
// that need to know which user the cookie maps to without forcing a 401.
func ResolveUploaderSession(ctx context.Context, db *sql.DB, r *http.Request) (*AuthUser, *model.UploaderSession, error) {
	return resolveUploaderSession(ctx, db, r)
}

func writeUploaderAuthError(w http.ResponseWriter, err error) {
	ae, ok := err.(*authError)
	if !ok {
		ae = &authError{http.StatusInternalServerError, "authentication error"}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(ae.Status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": ae.Message})
}
