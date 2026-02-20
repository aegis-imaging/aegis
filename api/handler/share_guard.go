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
