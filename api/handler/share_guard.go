package handler

import (
	"time"

	"github.com/msenjem/aegis/api/model"
)

// shareGoneMessage returns the public-facing reason when a share token is no
// longer usable. Empty string means the share is still active.
func shareGoneMessage(share *model.ExportShare, now time.Time) string {
	if share == nil {
		return "share not found"
	}
	if share.RevokedAt != nil {
		return "share has been revoked"
	}
	if now.UTC().After(share.ExpiresAt.UTC()) {
		return "share has expired"
	}
	return ""
}
