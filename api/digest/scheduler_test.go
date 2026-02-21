package digest

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPeriodRangeWeekly_UsesSingleUTCClock(t *testing.T) {
	now := time.Date(2026, 2, 20, 16, 45, 30, 0, time.FixedZone("CST", -6*3600))

	sinceUTC, untilUTC, label := periodRange("weekly", now)

	assert.Equal(t, time.UTC, sinceUTC.Location())
	assert.Equal(t, time.UTC, untilUTC.Location())
	assert.Equal(t, now.UTC(), untilUTC)
	assert.Equal(t, now.UTC().AddDate(0, 0, -7), sinceUTC)
	assert.Equal(t, "2026-02-13 22:45 UTC – 2026-02-20 22:45 UTC", label)
}

func TestPeriodRangeMonthly_UsesSingleUTCClock(t *testing.T) {
	now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	sinceUTC, untilUTC, label := periodRange("monthly", now)

	assert.Equal(t, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), sinceUTC)
	assert.Equal(t, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), untilUTC)
	assert.Equal(t, "2026-02-01 00:00 UTC – 2026-03-01 00:00 UTC", label)
}

func TestPeriodRangeMonthly_ClampsShortMonthBoundary(t *testing.T) {
	now := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)

	sinceUTC, untilUTC, label := periodRange("monthly", now)

	assert.Equal(t, time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC), sinceUTC)
	assert.Equal(t, time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC), untilUTC)
	assert.Equal(t, "2026-02-28 12:00 UTC – 2026-03-31 12:00 UTC", label)
}

func TestPeriodRange_DefaultsToWeekly(t *testing.T) {
	now := time.Date(2026, 6, 10, 9, 15, 0, 0, time.UTC)

	sinceUTC, untilUTC, _ := periodRange("unexpected", now)

	assert.Equal(t, now.UTC(), untilUTC)
	assert.Equal(t, now.UTC().AddDate(0, 0, -7), sinceUTC)
}

func TestIsDigestDue_NoLastSentIsDue(t *testing.T) {
	now := time.Date(2026, 6, 10, 9, 15, 0, 0, time.UTC)
	assert.True(t, isDigestDue("weekly", nil, now))
	assert.True(t, isDigestDue("monthly", nil, now))
}

func TestIsDigestDue_WeeklyUsesSevenDays(t *testing.T) {
	now := time.Date(2026, 6, 10, 9, 15, 0, 0, time.UTC)
	old := now.AddDate(0, 0, -8)
	recent := now.AddDate(0, 0, -6)
	boundary := now.AddDate(0, 0, -7)

	assert.True(t, isDigestDue("weekly", &old, now))
	assert.False(t, isDigestDue("weekly", &recent, now))
	assert.False(t, isDigestDue("weekly", &boundary, now))
}

func TestIsDigestDue_MonthlyUsesCalendarMonth(t *testing.T) {
	now := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)
	old := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
	recent := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	boundary := time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC)

	assert.True(t, isDigestDue("monthly", &old, now))
	assert.False(t, isDigestDue("monthly", &recent, now))
	assert.False(t, isDigestDue("monthly", &boundary, now))
}
