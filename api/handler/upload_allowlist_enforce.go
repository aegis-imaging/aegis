package handler

import (
	"log"
	"net/http"

	"github.com/aegis-imaging/aegis/api/model"
)

// enforceUploadMethod is the call-site check that institutions actually
// use to deny specific upload paths via their browser-upload allowlist.
// Call it from every upload handler that maps to one of
// model.KnownUploadMethods immediately after the caller's institution
// is attributed.
//
// Returns true when the caller should proceed. When it returns false,
// the response has already been written (403 for an explicit deny,
// 500 for a database error). Pass institutionID = nil when no
// institution could be attributed — that's treated as "default-on"
// per the design contract so legacy/anonymous flows keep working.
//
// The helper is intentionally minimal: no audit emission here (the
// caller already audits its own action), no telemetry, no per-method
// feature flag. Per the chunk-2 rollout plan, if a misconfigured row
// causes problems an admin can DELETE the explicit row through the
// existing /upload-allowlist API to revert that (institution, method)
// pair to default-on without a redeploy.
func (s *Server) enforceUploadMethod(w http.ResponseWriter, r *http.Request, institutionID *string, methodID string) bool {
	if institutionID == nil || *institutionID == "" {
		return true
	}
	allowed, err := model.IsUploadMethodAllowed(r.Context(), s.db, *institutionID, methodID)
	if err != nil {
		log.Printf("upload allowlist check (%s, %s): %v", *institutionID, methodID, err)
		s.writeError(w, http.StatusInternalServerError, "upload allowlist check failed")
		return false
	}
	if !allowed {
		s.writeError(w, http.StatusForbidden, "this upload method is disabled for your institution")
		return false
	}
	return true
}
