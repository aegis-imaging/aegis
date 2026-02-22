package email

import (
	"fmt"
	"strings"
	"time"
)

// StuckStudy is a minimal view used by the SLA alert email. No PHI included.
type StuckStudy struct {
	StudyInstanceUID string
	Status          string
	UpdatedAt       time.Time
}

// StudiesStuck generates the subject and plain-text body for an SLA stuck-study alert email.
// It lists each stuck study with its UID, status, and how long it has been idle.
// No PHI is included — only study UIDs and pipeline status.
func StudiesStuck(studies []StuckStudy, minutes int) (subject, body string) {
	subject = fmt.Sprintf("[AEGIS Alert] %d study/studies stuck in pipeline (>%d min idle)", len(studies), minutes)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("AEGIS Pipeline SLA Alert\n"))
	sb.WriteString(fmt.Sprintf("========================\n\n"))
	sb.WriteString(fmt.Sprintf("%d study/studies have been idle for more than %d minutes without reaching an\n", len(studies), minutes))
	sb.WriteString(fmt.Sprintf("approved or rejected terminal state. Please review the admin dashboard.\n\n"))
	sb.WriteString(fmt.Sprintf("Stuck Studies\n"))
	sb.WriteString(fmt.Sprintf("-------------\n"))
	for _, s := range studies {
		idle := time.Since(s.UpdatedAt).Round(time.Minute)
		sb.WriteString(fmt.Sprintf("  • UID: %s\n", s.StudyInstanceUID))
		sb.WriteString(fmt.Sprintf("    Status: %s  |  Idle: %s  |  Updated: %s UTC\n\n",
			s.Status, idle, s.UpdatedAt.UTC().Format("2006-01-02 15:04")))
	}
	sb.WriteString("\nThis alert will not repeat for studies already listed above until the cooldown\n")
	sb.WriteString("period has elapsed or the studies are approved/rejected.\n\n")
	sb.WriteString("— AEGIS automated alert\n")
	body = sb.String()
	return subject, body
}

