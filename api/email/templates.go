package email

import (
	"bytes"
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
