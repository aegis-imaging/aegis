package handler

import (
	"time"

	"github.com/msenjem/aegis/api/model"
)

// shareStatus returns a stable share status label evaluated with UTC semantics.
func shareStatus(share *model.ExportShare, now time.Time) string {
	if share == nil {
		return "unknown"
	}
	if share.RevokedAt != nil {
		return "revoked"
	}
	if now.UTC().After(share.ExpiresAt.UTC()) {
		return "expired"
	}
	return "active"
}

// shareExpiresInSeconds returns the remaining lifetime from now until expiry.
// Values are clamped at 0 once expired.
func shareExpiresInSeconds(share *model.ExportShare, now time.Time) int64 {
	if share == nil {
		return 0
	}
	remaining := share.ExpiresAt.UTC().Sub(now.UTC())
	if remaining <= 0 {
		return 0
	}
	return int64(remaining / time.Second)
}

// shareGoneMessage returns the public-facing reason when a share token is no
// longer usable. Empty string means the share is still active.
func shareGoneMessage(share *model.ExportShare, now time.Time) string {
	switch shareStatus(share, now) {
	case "unknown":
		return "share not found"
	case "revoked":
		return "share has been revoked"
	case "expired":
		return "share has expired"
	default:
		return ""
	}
}
