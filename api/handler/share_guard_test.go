package handler

import (
	"testing"
	"time"

	"github.com/msenjem/aegis/api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShareGoneMessage_Active(t *testing.T) {
	now := time.Date(2026, 2, 20, 12, 0, 0, 0, time.UTC)
	share := &model.ExportShare{
		ExpiresAt: now.Add(2 * time.Hour),
	}
	assert.Equal(t, "", shareGoneMessage(share, now))
}

func TestShareGoneMessage_Expired(t *testing.T) {
	now := time.Date(2026, 2, 20, 12, 0, 0, 0, time.UTC)
	share := &model.ExportShare{
		ExpiresAt: now.Add(-1 * time.Minute),
	}
	assert.Equal(t, "share has expired", shareGoneMessage(share, now))
}

func TestShareGoneMessage_ExpiredWithDifferentTimeZones(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	now := time.Date(2026, 2, 20, 12, 0, 0, 0, time.UTC)
	share := &model.ExportShare{
		// Same calendar day represented in EST, still an absolute UTC instant.
		ExpiresAt: time.Date(2026, 2, 20, 6, 59, 59, 0, loc),
	}
	assert.Equal(t, "share has expired", shareGoneMessage(share, now))
}

func TestShareGoneMessage_RevokedWinsOverExpiry(t *testing.T) {
	now := time.Date(2026, 2, 20, 12, 0, 0, 0, time.UTC)
	revokedAt := now.Add(-3 * time.Hour)
	share := &model.ExportShare{
		ExpiresAt: now.Add(-1 * time.Hour),
		RevokedAt: &revokedAt,
	}
	assert.Equal(t, "share has been revoked", shareGoneMessage(share, now))
}
