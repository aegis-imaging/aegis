package email

import (
	"bytes"
	"strings"
	"text/template"
	"time"
)

var shareCreatedTmpl = template.Must(template.New("share_created").Parse(
	`You have been granted access to a de-identified medical imaging study through the
Anonymization & Exchange Gateway for Imaging Studies (AEGIS).

Click the link below to download the anonymized DICOM files:

  {{ .ExportURL }}

This link expires on {{ .ExpiresAt }}.
{{ if .Note }}
Note from the sender: {{ .Note }}
{{ end }}
If you have questions, please contact the sender directly.

--
This is an automated message from AEGIS. Do not reply to this email.
`))

var uploadConfirmedTmpl = template.Must(template.New("upload_confirmed").Parse(
	`Your DICOM upload has been received and is queued for review.

Study UID:  {{ .StudyUID }}
Modality:   {{ .Modality }}
Files:      {{ .FileCount }}
Received:   {{ .ReceivedAt }}

You will be notified once your study has been reviewed by an administrator.

--
This is an automated message from AEGIS. Do not reply to this email.
`))

var studyApprovedTmpl = template.Must(template.New("study_approved").Parse(
	`Your study submission has been reviewed and approved.

Study UID:  {{ .StudyUID }}

The study is now eligible for sharing with collaborators.

--
This is an automated message from AEGIS. Do not reply to this email.
`))

var studyRejectedTmpl = template.Must(template.New("study_rejected").Parse(
	`Your study submission has been reviewed and rejected.

Study UID:  {{ .StudyUID }}

Please contact the administrator for more information.

--
This is an automated message from AEGIS. Do not reply to this email.
`))

func ShareCreated(exportURL string, expiresAt time.Time, note string) (subject, body string) {
	subject = "AEGIS — Imaging Study Access"
	var buf bytes.Buffer
	shareCreatedTmpl.Execute(&buf, struct {
		ExportURL string
		ExpiresAt string
		Note      string
	}{exportURL, expiresAt.UTC().Format("2006-01-02 15:04 UTC"), note})
	return subject, buf.String()
}

func UploadConfirmed(studyUID, modality string, fileCount int, receivedAt time.Time) (subject, body string) {
	subject = "AEGIS — Upload Received"
	var buf bytes.Buffer
	uploadConfirmedTmpl.Execute(&buf, struct {
		StudyUID   string
		Modality   string
		FileCount  int
		ReceivedAt string
	}{studyUID, modality, fileCount, receivedAt.UTC().Format("2006-01-02 15:04 UTC")})
	return subject, buf.String()
}

func StudyApproved(studyUID string) (subject, body string) {
	subject = "AEGIS — Study Approved"
	var buf bytes.Buffer
	studyApprovedTmpl.Execute(&buf, struct{ StudyUID string }{studyUID})
	return subject, buf.String()
}

func StudyRejected(studyUID string) (subject, body string) {
	subject = "AEGIS — Study Rejected"
	var buf bytes.Buffer
	studyRejectedTmpl.Execute(&buf, struct{ StudyUID string }{studyUID})
	return subject, buf.String()
}

var digestSummaryTmpl = template.Must(template.New("digest_summary").Parse(
	`AEGIS {{ .FrequencyTitle }} Summary — {{ .ProjectName }} — {{ .PeriodLabel }}

Studies
  Received this period:  {{ .Received }}
  Approved:              {{ .Approved }}
  Rejected:              {{ .Rejected }}
  Pending review:        {{ .Pending }}

Export shares created:   {{ .SharesCreated }}

--
This is an automated message from AEGIS. To unsubscribe, contact your administrator.
`))

var pipelineFailureTmpl = template.Must(template.New("pipeline_failure").Parse(
	`An automated pipeline step has failed for a study in AEGIS.

Service:    {{ .Service }}
Study UID:  {{ .StudyUID }}
Failed at:  {{ .FailedAt }}

Error:
  {{ .ErrMsg }}

Please review the study in the admin dashboard and re-trigger the step or
investigate the sidecar service logs.

--
This is an automated alert from AEGIS. Do not reply to this email.
`))

// PipelineFailure renders an alert email for a pipeline service step failure.
// No PHI is included — only the study UID, service name, and error message.
func PipelineFailure(studyUID, service, errMsg string) (subject, body string) {
	subject = "[AEGIS Alert] Pipeline step failed: " + service + " — " + studyUID
	var buf bytes.Buffer
	pipelineFailureTmpl.Execute(&buf, struct {
		Service  string
		StudyUID string
		FailedAt string
		ErrMsg   string
	}{service, studyUID, time.Now().UTC().Format("2006-01-02 15:04:05 UTC"), errMsg})
	return subject, buf.String()
}

var contactFormTmpl = template.Must(template.New("contact_form").Parse(
	`New contact form submission from the AEGIS website.

Name:         {{ .Name }}
Email:        {{ .Email }}
{{ if .Organization }}Organization: {{ .Organization }}
{{ end }}{{ if .Role }}Role:         {{ .Role }}
{{ end }}
Message:
{{ .Message }}

--
This message was submitted via the contact form at aegisimaging.ai.
`))

// ContactForm renders a contact form submission email.
func ContactForm(name, contactEmail, organization, role, message string) (subject, body string) {
	subject = "[AEGIS Contact] " + name + " — " + truncate(message, 60)
	var buf bytes.Buffer
	contactFormTmpl.Execute(&buf, struct {
		Name         string
		Email        string
		Organization string
		Role         string
		Message      string
	}{name, contactEmail, organization, role, message})
	return subject, buf.String()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// DigestSummary renders a periodic digest email for a project.
func DigestSummary(projectName, frequency, periodLabel string, received, approved, rejected, pending, sharesCreated int) (subject, body string) {
	freqTitle := strings.ToUpper(frequency[:1]) + frequency[1:]
	subject = "AEGIS " + freqTitle + " Summary — " + projectName
	var buf bytes.Buffer
	digestSummaryTmpl.Execute(&buf, struct {
		FrequencyTitle string
		ProjectName    string
		PeriodLabel    string
		Received       int
		Approved       int
		Rejected       int
		Pending        int
		SharesCreated  int
	}{freqTitle, projectName, periodLabel, received, approved, rejected, pending, sharesCreated})
	return subject, buf.String()
}
