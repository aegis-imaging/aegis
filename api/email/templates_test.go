package email

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestShareCreated(t *testing.T) {
	url := "https://aegis.example.com/export/abc123"
	expires := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	subject, body := ShareCreated(url, expires, "Please review")

	assert.Equal(t, "AEGIS — Imaging Study Access", subject)
	assert.Contains(t, body, url)
	assert.Contains(t, body, "2026-03-01 12:00 UTC")
	assert.Contains(t, body, "Please review")
}

func TestShareCreated_NoNote(t *testing.T) {
	url := "https://aegis.example.com/export/abc123"
	expires := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	_, body := ShareCreated(url, expires, "")

	assert.Contains(t, body, url)
	assert.NotContains(t, body, "Note from the sender:")
}

func TestUploadConfirmed(t *testing.T) {
	received := time.Date(2026, 2, 19, 10, 30, 0, 0, time.UTC)
	subject, body := UploadConfirmed("1.2.3.4", "MRI", 42, received)

	assert.Equal(t, "AEGIS — Upload Received", subject)
	assert.Contains(t, body, "1.2.3.4")
	assert.Contains(t, body, "MRI")
	assert.Contains(t, body, "42")
	assert.Contains(t, body, "2026-02-19 10:30 UTC")
}

func TestStudyApproved(t *testing.T) {
	subject, body := StudyApproved("1.2.3.4")

	assert.Equal(t, "AEGIS — Study Approved", subject)
	assert.Contains(t, body, "1.2.3.4")
	assert.Contains(t, body, "approved")
}

func TestStudyRejected(t *testing.T) {
	subject, body := StudyRejected("1.2.3.4", "")

	assert.Equal(t, "AEGIS — Study Rejected", subject)
	assert.Contains(t, body, "1.2.3.4")
	assert.Contains(t, body, "rejected")
}

func TestStudyRejected_WithReason(t *testing.T) {
	_, body := StudyRejected("1.2.3.4", "Poor image quality")

	assert.Contains(t, body, "1.2.3.4")
	assert.Contains(t, body, "Poor image quality")
}

func TestDigestSummary(t *testing.T) {
	subject, body := DigestSummary("Brain Study", "weekly", "Feb 12 – Feb 19", 10, 5, 2, 3, 1)

	assert.Equal(t, "AEGIS Weekly Summary — Brain Study", subject)
	assert.Contains(t, body, "Brain Study")
	assert.Contains(t, body, "Weekly")
	assert.Contains(t, body, "Feb 12 – Feb 19")
	assert.Contains(t, body, "10")
	assert.Contains(t, body, "5")
	assert.Contains(t, body, "2")
	assert.Contains(t, body, "3")
	assert.Contains(t, body, "1")
}

func TestDigestSummary_Monthly(t *testing.T) {
	subject, _ := DigestSummary("Default", "monthly", "Jan 2026", 0, 0, 0, 0, 0)
	assert.Equal(t, "AEGIS Monthly Summary — Default", subject)
}

func TestTemplates_NoPHI(t *testing.T) {
	// Verify no template includes real patient data patterns
	_, shareBody := ShareCreated("https://example.com", time.Now(), "test")
	_, uploadBody := UploadConfirmed("1.2.3", "CT", 1, time.Now())
	_, approvedBody := StudyApproved("1.2.3")
	_, rejectedBody := StudyRejected("1.2.3", "")
	_, digestBody := DigestSummary("Test", "weekly", "test", 0, 0, 0, 0, 0)

	for _, body := range []string{shareBody, uploadBody, approvedBody, rejectedBody, digestBody} {
		assert.NotContains(t, body, "patient")
		assert.NotContains(t, body, "Patient")
	}
}
