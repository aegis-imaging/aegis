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
{{ if .ProjectName }}
Project:    {{ .ProjectName }}
{{ end }}
You will be notified once your study has been reviewed by an administrator.

--
This is an automated message from AEGIS. Do not reply to this email.
`))

var studyApprovedTmpl = template.Must(template.New("study_approved").Parse(
	`Your study submission has been reviewed and approved.

Study UID:  {{ .StudyUID }}
{{ if .ProjectName }}Project:    {{ .ProjectName }}
{{ end }}
The study is now eligible for sharing with collaborators.

--
This is an automated message from AEGIS. Do not reply to this email.
`))

var studyRejectedTmpl = template.Must(template.New("study_rejected").Parse(
	`Your study submission has been reviewed and rejected.

Study UID:  {{ .StudyUID }}
{{ if .ProjectName }}Project:    {{ .ProjectName }}
{{ end }}{{ if .Reason }}Reason:     {{ .Reason }}
{{ end }}
Please contact your administrator if you have questions about this decision.

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

func UploadConfirmed(studyUID, modality, projectName string, fileCount int, receivedAt time.Time) (subject, body string) {
	subject = "AEGIS — Upload Received"
	var buf bytes.Buffer
	uploadConfirmedTmpl.Execute(&buf, struct {
		StudyUID    string
		Modality    string
		ProjectName string
		FileCount   int
		ReceivedAt  string
	}{studyUID, modality, projectName, fileCount, receivedAt.UTC().Format("2006-01-02 15:04 UTC")})
	return subject, buf.String()
}

func StudyApproved(studyUID, projectName string) (subject, body string) {
	subject = "AEGIS — Study Approved"
	var buf bytes.Buffer
	studyApprovedTmpl.Execute(&buf, struct {
		StudyUID    string
		ProjectName string
	}{studyUID, projectName})
	return subject, buf.String()
}

func StudyRejected(studyUID, reason, projectName string) (subject, body string) {
	subject = "AEGIS — Study Rejected"
	var buf bytes.Buffer
	studyRejectedTmpl.Execute(&buf, struct {
		StudyUID    string
		Reason      string
		ProjectName string
	}{studyUID, reason, projectName})
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

var destinationFailingTmpl = template.Must(template.New("destination_failing").Parse(
`A DICOM destination has stopped responding.

Destination: {{ .Name }} ({{ .Type }})
Error:        {{ .ErrMsg }}
Detected at:  {{ .DetectedAt }} UTC

The destination health probe recorded a failure. Check that the remote
endpoint is online and that any firewall rules permit outbound connections.

--
This is an automated alert from AEGIS. Do not reply to this email.
`))

// DestinationFailing renders an alert email when a DICOM destination transitions to failing.
// No PHI is included — only the destination name, type, and error message.
func DestinationFailing(name, destType, errMsg string) (subject, body string) {
	subject = "[AEGIS Alert] Destination unreachable: " + name
	var buf bytes.Buffer
	destinationFailingTmpl.Execute(&buf, struct {
		Name        string
		Type        string
		ErrMsg      string
		DetectedAt  string
	}{name, destType, errMsg, time.Now().UTC().Format("2006-01-02 15:04:05 UTC")})
	return subject, buf.String()
}

var inviteRequestTmpl = template.Must(template.New("invite_request").Parse(
	`Someone has requested early access to AEGIS.

Name:         {{ .Name }}
Email:        {{ .Email }}
{{ if .Organization }}Organization: {{ .Organization }}
{{ end }}{{ if .Message }}Message:
{{ .Message }}

{{ end }}Requested at: {{ .RequestedAt }} UTC

Click the link below to instantly generate an invite code and email it to them:

  {{ .ApprovalURL }}

This link is valid for 7 days. You can also create codes manually in the admin dashboard.

--
This is an automated notification from AEGIS.
`))

var inviteCodeIssuedTmpl = template.Must(template.New("invite_code_issued").Parse(
	`Hi {{ .Name }},

You've been granted early access to AEGIS — the HIPAA-compliant medical imaging platform.

Your invite code is:

  {{ .Code }}

Use the link below to access AEGIS with your code pre-filled:

  {{ .InviteURL }}

Or enter the code manually at {{ .LandingURL }}.

--
This is an automated message from AEGIS. Do not reply to this email.
`))

// InviteRequest renders the admin notification email sent when someone requests access.
func InviteRequest(name, requestEmail, organization, message, approvalURL string) (subject, body string) {
	subject = "[AEGIS] Access request from " + name
	var buf bytes.Buffer
	inviteRequestTmpl.Execute(&buf, struct {
		Name        string
		Email       string
		Organization string
		Message     string
		ApprovalURL string
		RequestedAt string
	}{name, requestEmail, organization, message, approvalURL, time.Now().UTC().Format("2006-01-02 15:04:05")})
	return subject, buf.String()
}

// InviteCodeIssued renders the email sent to the requester with their invite code.
func InviteCodeIssued(name, code, inviteURL, landingURL string) (subject, body string) {
	subject = "Your AEGIS invite code"
	var buf bytes.Buffer
	inviteCodeIssuedTmpl.Execute(&buf, struct {
		Name       string
		Code       string
		InviteURL  string
		LandingURL string
	}{name, code, inviteURL, landingURL})
	return subject, buf.String()
}

var adminDashboardInviteTmpl = template.Must(template.New("admin_dashboard_invite").Parse(
	`Hi,

You have been granted {{ .Role }} access to the AEGIS Admin Dashboard.

Dashboard URL:

  {{ .DashboardURL }}

Sign in using your existing Google, Microsoft, or AWS account at the link above.
No separate password is required — use your organisation's single sign-on.

Your role: {{ .Role }}

--
This is an automated message from AEGIS. Do not reply to this email.
`))

// AdminDashboardInvite renders the welcome email sent when an admin invites a user
// to the dashboard directly. No PHI is included.
func AdminDashboardInvite(toEmail, dashboardURL, role string) (subject, body string) {
	subject = "You've been invited to the AEGIS Admin Dashboard"
	var buf bytes.Buffer
	adminDashboardInviteTmpl.Execute(&buf, struct {
		DashboardURL string
		Role         string
	}{dashboardURL, role})
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

var desktopInstallerInviteTmpl = template.Must(template.New("desktop_installer_invite").Parse(
	`Hi {{ .Name }},

You have been sent a download link for {{ .ProductLabel }} ({{ .PlatformLabel }}).

Download:

  {{ .InstallURL }}

This link expires on {{ .ExpiresAt }}. If it expires before you install, contact
your AEGIS administrator and they can resend the link.

After installing, the app will pair with your AEGIS account automatically the
first time you launch it.

--
This is an automated message from AEGIS. Do not reply to this email.
`))

// DesktopInstallerInvite renders the email sent to a recipient when an admin
// dispatches a desktop client install link from the admin dashboard. The body
// includes the time-limited install URL — never the raw pairing token.
func DesktopInstallerInvite(name, productLabel, platformLabel, installURL string, expiresAt time.Time) (subject, body string) {
	if name == "" {
		name = "there"
	}
	subject = "[AEGIS] Your " + productLabel + " download link"
	var buf bytes.Buffer
	desktopInstallerInviteTmpl.Execute(&buf, struct {
		Name          string
		ProductLabel  string
		PlatformLabel string
		InstallURL    string
		ExpiresAt     string
	}{name, productLabel, platformLabel, installURL,
		expiresAt.UTC().Format("2006-01-02 15:04 UTC")})
	return subject, buf.String()
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

var uploaderInviteTmpl = template.Must(template.New("uploader_invite").Parse(
	`Hi {{ .Name }},

You've been invited to contribute imaging studies to the project
"{{ .ProjectName }}" in AEGIS.

Click the link below to set a password and start uploading:

  {{ .RedeemURL }}

This invitation expires on {{ .ExpiresAt }}.

After you set your password you'll be able to sign in at the upload
portal whenever you have new data to contribute. You'll only have
access to upload to "{{ .ProjectName }}" — nothing else.

--
This is an automated message from AEGIS. Do not reply to this email.
`))

// UploaderInvite renders the email sent to an outside contributor when an
// admin or researcher invites them to upload to a specific project. The body
// includes the time-limited redeem URL — never the raw token alone.
func UploaderInvite(name, projectName, redeemURL string, expiresAt time.Time) (subject, body string) {
	if name == "" {
		name = "there"
	}
	subject = "[AEGIS] Invitation to upload to " + projectName
	var buf bytes.Buffer
	uploaderInviteTmpl.Execute(&buf, struct {
		Name        string
		ProjectName string
		RedeemURL   string
		ExpiresAt   string
	}{name, projectName, redeemURL,
		expiresAt.UTC().Format("2006-01-02 15:04 UTC")})
	return subject, buf.String()
}
