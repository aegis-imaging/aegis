package handler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveShareExpiry_DefaultSevenDays(t *testing.T) {
	now := time.Date(2026, 2, 20, 12, 0, 0, 0, time.UTC)

	expiresAt, err := resolveShareExpiry(createShareRequest{}, now)
	require.NoError(t, err)
	assert.Equal(t, now.Add(168*time.Hour), expiresAt)
}

func TestResolveShareExpiry_UsesExpiryHours(t *testing.T) {
	now := time.Date(2026, 2, 20, 12, 0, 0, 0, time.UTC)

	expiresAt, err := resolveShareExpiry(createShareRequest{ExpiryHours: 48}, now)
	require.NoError(t, err)
	assert.Equal(t, now.Add(48*time.Hour), expiresAt)
}

func TestResolveShareExpiry_UsesExplicitExpiresAt(t *testing.T) {
	now := time.Date(2026, 2, 20, 12, 0, 0, 0, time.UTC)
	req := createShareRequest{
		ExpiryHours: 24, // ignored when expires_at is provided
		ExpiresAt:   "2026-02-22T07:30:00-05:00",
	}

	expiresAt, err := resolveShareExpiry(req, now)
	require.NoError(t, err)
	assert.Equal(t, "2026-02-22T12:30:00Z", expiresAt.Format(time.RFC3339))
}

func TestResolveShareExpiry_RejectsInvalidExpiresAt(t *testing.T) {
	now := time.Date(2026, 2, 20, 12, 0, 0, 0, time.UTC)

	_, err := resolveShareExpiry(createShareRequest{ExpiresAt: "02/22/2026 12:00"}, now)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RFC3339")
}

func TestResolveShareExpiry_RejectsPastExpiresAt(t *testing.T) {
	now := time.Date(2026, 2, 20, 12, 0, 0, 0, time.UTC)

	_, err := resolveShareExpiry(createShareRequest{ExpiresAt: "2026-02-19T12:00:00Z"}, now)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "future")
}
