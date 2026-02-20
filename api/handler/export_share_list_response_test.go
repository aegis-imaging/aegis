package handler

import (
	"testing"
	"time"

	"github.com/msenjem/aegis/api/model"
	"github.com/stretchr/testify/assert"
)

func TestBuildListShareResponses_IncludesStatusAndRemainingSeconds(t *testing.T) {
	now := time.Date(2026, 2, 20, 12, 0, 0, 0, time.UTC)
	revokedAt := now.Add(-5 * time.Minute)

	shares := []model.ExportShare{
		{ID: "active", ExpiresAt: now.Add(2 * time.Minute)},
		{ID: "expired", ExpiresAt: now.Add(-10 * time.Second)},
		{ID: "revoked", ExpiresAt: now.Add(30 * time.Minute), RevokedAt: &revokedAt},
	}

	out := buildListShareResponses(shares, now)
	assert.Len(t, out, 3)

	byID := map[string]listShareResponse{}
	for _, s := range out {
		byID[s.ID] = s
	}

	assert.Equal(t, "active", byID["active"].Status)
	assert.EqualValues(t, 120, byID["active"].ExpiresInSeconds)

	assert.Equal(t, "expired", byID["expired"].Status)
	assert.EqualValues(t, 0, byID["expired"].ExpiresInSeconds)

	assert.Equal(t, "revoked", byID["revoked"].Status)
	assert.EqualValues(t, 1800, byID["revoked"].ExpiresInSeconds)
}
