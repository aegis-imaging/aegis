// Upload auth middleware — gates the public /api/upload/* routes when
// AUTH_ENABLED=true. PR #566 shipped uploader accounts but deliberately left
// the upload routes open; this middleware closes that gap without breaking
// any of the three existing upload identities:
//
//  1. Satellite routers — WithSatelliteMTLS runs first in the chain and puts a
//     SatelliteIdentity on ctx when a known client cert is presented. The cert
//     is the credential; no session or key is needed on top.
//  2. Upload-portal users — the aegis_uploader_session cookie set by
//     uploader-login / invite redemption. Per-project authorization for
//     UploadInit happens in the handler (it needs the resolved project).
//  3. API keys — Authorization: Bearer <key>, the same path desktop-app
//     pairing produces. Reuses apiKeyUser from auth.go.
//
// When cfg.AuthEnabled is false (local dev / docker-compose) the middleware is
// a strict no-op so a zero-account local deployment keeps working.
package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/config"
	"github.com/aegis-imaging/aegis/api/model"
)

// RequireUploadAuth enforces upload authentication as described above. It must
// be wrapped INSIDE WithSatelliteMTLS (i.e. satelliteMTLS(uploadAuth(handler)))
// so the satellite identity, when present, is already on the context.
func RequireUploadAuth(db *sql.DB, cfg *config.Config) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if !cfg.AuthEnabled {
				next(w, r)
				return
			}

			// (a) Satellite mTLS identity — strongest signal, checked first.
			if SatelliteFromContext(r.Context()) != nil {
				next(w, r)
				return
			}

			// (b) Uploader session cookie. An invalid/expired cookie falls
			// through to the API-key check so a desktop app that happens to
			// carry a stale browser cookie isn't locked out of its key auth.
			if c, err := r.Cookie(UploaderSessionCookieName); err == nil && c.Value != "" {
				user, sess, serr := resolveUploaderSession(r.Context(), db, r)
				if serr == nil {
					sessID := sess.ID
					go func() {
						_ = model.TouchUploaderSession(context.Background(), db, sessID)
					}()
					ctx := context.WithValue(r.Context(), authUserKey, user)
					ctx = context.WithValue(ctx, uploaderUserKey{}, sess)
					next(w, r.WithContext(ctx))
					return
				}
			}

			// (c) API key via Authorization: Bearer.
			if strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
				user, kerr := apiKeyUser(r.Context(), db, r)
				if kerr != nil {
					writeAuthError(w, kerr)
					return
				}
				ctx := context.WithValue(r.Context(), authUserKey, user)
				next(w, r.WithContext(ctx))
				return
			}

			writeUploaderAuthError(w, &authError{
				http.StatusUnauthorized,
				"authentication required: sign in to the upload portal or supply an API key",
			})
		}
	}
}
