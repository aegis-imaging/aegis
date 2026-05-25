package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/aegis-imaging/aegis/api/email"
	"github.com/aegis-imaging/aegis/api/model"
)

// Allowed values for (product, platform). Kept as small allow-lists rather
// than free text so the admin UI's selector and the binary upload form stay
// in sync, and so we can render readable labels in emails.
var allowedInstallerProducts = map[string]string{
	"uploader":      "AEGIS Desktop Uploader",
	"dimse-bridge":  "AEGIS DIMSE Bridge",
}

var allowedInstallerPlatforms = map[string]string{
	"macos-arm64":     "macOS (Apple Silicon)",
	"macos-x64":       "macOS (Intel)",
	"windows-x64":     "Windows (64-bit)",
	"linux-deb":       "Linux (.deb)",
	"linux-appimage":  "Linux (AppImage)",
	"linux-rpm":       "Linux (.rpm)",
}

// Max upload size: 500 MiB. Tauri uploader binaries are ~10 MB; PyInstaller
// bundles for the DIMSE bridge can reach ~150 MB on macOS once numpy + pylibjpeg
// are bundled. 500 MiB leaves headroom without inviting abuse.
const maxInstallerUploadBytes int64 = 500 << 20

// semverPattern is a permissive semver matcher (e.g. 1.2.3, 1.2.3-beta.1, 1.2.3+build).
// Not RFC-strict; we just want to reject obviously bad versions like "../oops".
var semverPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`)

// filenameSafePattern allows letters, digits, dot, dash, underscore. We never
// echo a user-supplied filename into a path without re-deriving it server-side,
// but the upload form's filename ends up in the Content-Disposition header.
var filenameSafePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// validProductPlatform returns an error if (product, platform) isn't an allowed combo.
// Currently every product allows every platform — the allow-list lives in two
// separate maps so a future product can constrain platforms without breaking this
// function's call sites.
func validProductPlatform(product, platform string) error {
	if _, ok := allowedInstallerProducts[product]; !ok {
		return fmt.Errorf("unknown product %q", product)
	}
	if _, ok := allowedInstallerPlatforms[platform]; !ok {
		return fmt.Errorf("unknown platform %q", platform)
	}
	return nil
}

// ListDesktopInstallers GET /api/desktop-installers
//
// Returns all installer rows (newest first per product/platform), so the
// admin UI can show version history alongside current.
func (s *Server) ListDesktopInstallers(w http.ResponseWriter, r *http.Request) {
	out, err := model.ListDesktopInstallers(r.Context(), s.db)
	if err != nil {
		log.Printf("list desktop installers: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to list installers")
		return
	}
	if out == nil {
		out = []model.DesktopInstaller{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"installers": out})
}

// CreateDesktopInstaller POST /api/desktop-installers
//
// Two modes:
//   - Multipart upload (Content-Type: multipart/form-data) with a 'file'
//     field. The handler streams the file to the configured Storage backend
//     under installers/{product}/{version}/{filename} and stores sha256.
//   - JSON body with external_url set. Skips storage entirely and just
//     records the row pointing at the external URL (e.g. GitHub Releases).
//
// All variants are admin-only.
func (s *Server) CreateDesktopInstaller(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/json") {
		s.createDesktopInstallerExternal(w, r)
		return
	}
	s.createDesktopInstallerUpload(w, r)
}

func (s *Server) createDesktopInstallerExternal(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Product     string `json:"product"`
		Platform    string `json:"platform"`
		Version     string `json:"version"`
		ExternalURL string `json:"external_url"`
		Filename    string `json:"filename"`
		SizeBytes   int64  `json:"size_bytes"`
		SHA256      string `json:"sha256"`
		Changelog   string `json:"changelog"`
		MarkCurrent bool   `json:"mark_current"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := validProductPlatform(req.Product, req.Platform); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !semverPattern.MatchString(req.Version) {
		s.writeError(w, http.StatusBadRequest, "version must be semver (e.g. 1.0.0)")
		return
	}
	if req.ExternalURL == "" {
		s.writeError(w, http.StatusBadRequest, "external_url required")
		return
	}
	if _, err := url.Parse(req.ExternalURL); err != nil {
		s.writeError(w, http.StatusBadRequest, "external_url is not a valid URL")
		return
	}
	if req.Filename == "" {
		s.writeError(w, http.StatusBadRequest, "filename required")
		return
	}
	if !filenameSafePattern.MatchString(req.Filename) {
		s.writeError(w, http.StatusBadRequest, "filename contains disallowed characters")
		return
	}

	inst, err := model.CreateDesktopInstaller(r.Context(), s.db, model.CreateDesktopInstallerInput{
		Product:     req.Product,
		Platform:    req.Platform,
		Version:     req.Version,
		ExternalURL: req.ExternalURL,
		Filename:    req.Filename,
		SizeBytes:   req.SizeBytes,
		SHA256:      req.SHA256,
		Changelog:   req.Changelog,
		CreatedBy:   actorEmail(r),
	})
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			s.writeError(w, http.StatusConflict, "this product/platform/version already exists")
			return
		}
		log.Printf("create installer (external): %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to create installer")
		return
	}
	if req.MarkCurrent {
		if err := model.SetDesktopInstallerCurrent(r.Context(), s.db, inst.ID); err != nil {
			log.Printf("set installer current: %v", err)
			// Created OK — surface the row but warn.
		}
	}
	model.CreateAuditEntry(r.Context(), s.db, "desktop_installer.created", actorEmail(r),
		"desktop_installer", inst.ID, clientIP(r), map[string]any{
			"product": inst.Product, "platform": inst.Platform, "version": inst.Version,
			"external_url": inst.ExternalURL,
		})
	s.writeJSON(w, http.StatusCreated, inst)
}

func (s *Server) createDesktopInstallerUpload(w http.ResponseWriter, r *http.Request) {
	// 32 MB in-memory limit; everything else spills to a tempfile via the
	// multipart implementation. Total cap enforced by MaxBytesReader.
	r.Body = http.MaxBytesReader(w, r.Body, maxInstallerUploadBytes+(1<<20))
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		s.writeError(w, http.StatusBadRequest, "failed to parse upload: "+err.Error())
		return
	}

	product := r.FormValue("product")
	platform := r.FormValue("platform")
	version := r.FormValue("version")
	changelog := r.FormValue("changelog")
	markCurrent := r.FormValue("mark_current") == "true"

	if err := validProductPlatform(product, platform); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !semverPattern.MatchString(version) {
		s.writeError(w, http.StatusBadRequest, "version must be semver (e.g. 1.0.0)")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "missing 'file' field")
		return
	}
	defer file.Close()

	if header.Size > maxInstallerUploadBytes {
		s.writeError(w, http.StatusRequestEntityTooLarge,
			fmt.Sprintf("file too large (max %d bytes)", maxInstallerUploadBytes))
		return
	}
	filename := header.Filename
	if filename == "" || !filenameSafePattern.MatchString(filename) {
		s.writeError(w, http.StatusBadRequest, "filename missing or contains disallowed characters")
		return
	}

	// Stream the upload to a temp file while computing sha256. Storage.Store
	// only takes io.Reader; we can't both upload and hash from the same stream
	// without buffering somewhere. A tempfile bounds memory at the OS level.
	tmp, err := os.CreateTemp("", "aegis-installer-*.bin")
	if err != nil {
		log.Printf("create installer tempfile: %v", err)
		s.writeError(w, http.StatusInternalServerError, "tempfile failed")
		return
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(tmp, hasher), file)
	if err != nil {
		log.Printf("copy installer to tempfile: %v", err)
		s.writeError(w, http.StatusInternalServerError, "upload read failed")
		return
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		s.writeError(w, http.StatusInternalServerError, "tempfile seek failed")
		return
	}

	storageKey := fmt.Sprintf("installers/%s/%s/%s", product, version, filename)
	if err := s.store.Store(r.Context(), storageKey, tmp); err != nil {
		log.Printf("store installer to backend: %v", err)
		s.writeError(w, http.StatusInternalServerError, "storage write failed")
		return
	}

	inst, err := model.CreateDesktopInstaller(r.Context(), s.db, model.CreateDesktopInstallerInput{
		Product:    product,
		Platform:   platform,
		Version:    version,
		StorageKey: storageKey,
		Filename:   filename,
		SizeBytes:  written,
		SHA256:     hex.EncodeToString(hasher.Sum(nil)),
		Changelog:  changelog,
		CreatedBy:  actorEmail(r),
	})
	if err != nil {
		// Clean up the orphaned storage object before bailing.
		if delErr := s.store.Delete(r.Context(), storageKey); delErr != nil {
			log.Printf("cleanup orphaned installer object %s: %v", storageKey, delErr)
		}
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			s.writeError(w, http.StatusConflict, "this product/platform/version already exists")
			return
		}
		log.Printf("insert installer row: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to create installer")
		return
	}
	if markCurrent {
		if err := model.SetDesktopInstallerCurrent(r.Context(), s.db, inst.ID); err != nil {
			log.Printf("set installer current: %v", err)
		} else {
			inst.IsCurrent = true
		}
	}
	model.CreateAuditEntry(r.Context(), s.db, "desktop_installer.uploaded", actorEmail(r),
		"desktop_installer", inst.ID, clientIP(r), map[string]any{
			"product": inst.Product, "platform": inst.Platform, "version": inst.Version,
			"filename": inst.Filename, "size_bytes": inst.SizeBytes, "sha256": inst.SHA256,
		})
	s.writeJSON(w, http.StatusCreated, inst)
}

// UpdateDesktopInstaller PATCH /api/desktop-installers/{id}
//
// Accepts a partial JSON body with any of:
//   - "mark_current": true       — sets this version current (clears siblings)
//   - "changelog": "..."         — replaces the changelog text
func (s *Server) UpdateDesktopInstaller(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		MarkCurrent *bool   `json:"mark_current,omitempty"`
		Changelog   *string `json:"changelog,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.Changelog != nil {
		if err := model.UpdateDesktopInstallerChangelog(r.Context(), s.db, id, *req.Changelog); err != nil {
			log.Printf("update changelog: %v", err)
			s.writeError(w, http.StatusInternalServerError, "failed to update changelog")
			return
		}
		model.CreateAuditEntry(r.Context(), s.db, "desktop_installer.changelog_updated", actorEmail(r),
			"desktop_installer", id, clientIP(r), nil)
	}
	if req.MarkCurrent != nil && *req.MarkCurrent {
		if err := model.SetDesktopInstallerCurrent(r.Context(), s.db, id); err != nil {
			log.Printf("mark current: %v", err)
			s.writeError(w, http.StatusInternalServerError, "failed to mark current")
			return
		}
		model.CreateAuditEntry(r.Context(), s.db, "desktop_installer.marked_current", actorEmail(r),
			"desktop_installer", id, clientIP(r), nil)
	}

	inst, err := model.GetDesktopInstaller(r.Context(), s.db, id)
	if err != nil {
		if err == sql.ErrNoRows {
			s.writeError(w, http.StatusNotFound, "installer not found")
			return
		}
		log.Printf("get installer: %v", err)
		s.writeError(w, http.StatusInternalServerError, "fetch failed")
		return
	}
	s.writeJSON(w, http.StatusOK, inst)
}

// DeleteDesktopInstaller DELETE /api/desktop-installers/{id}
//
// Removes the DB row and best-effort deletes the storage object. Storage
// cleanup failures are logged but don't fail the request — the row is gone,
// and orphaned binaries can be GC'd separately.
func (s *Server) DeleteDesktopInstaller(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	inst, err := model.GetDesktopInstaller(r.Context(), s.db, id)
	if err != nil {
		if err == sql.ErrNoRows {
			s.writeError(w, http.StatusNotFound, "installer not found")
			return
		}
		log.Printf("get installer for delete: %v", err)
		s.writeError(w, http.StatusInternalServerError, "fetch failed")
		return
	}
	if err := model.DeleteDesktopInstaller(r.Context(), s.db, id); err != nil {
		log.Printf("delete installer row: %v", err)
		s.writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	if inst.StorageKey != "" {
		if err := s.store.Delete(r.Context(), inst.StorageKey); err != nil {
			log.Printf("delete installer object %s: %v", inst.StorageKey, err)
		}
	}
	model.CreateAuditEntry(r.Context(), s.db, "desktop_installer.deleted", actorEmail(r),
		"desktop_installer", id, clientIP(r), map[string]any{
			"product": inst.Product, "platform": inst.Platform, "version": inst.Version,
		})
	w.WriteHeader(http.StatusNoContent)
}

// DownloadDesktopInstaller GET /api/desktop-installers/{id}/download
//
// Authenticated self-serve. Redirects to a fresh 1-hour signed URL (or to the
// external_url if the installer isn't hosted by AEGIS).
func (s *Server) DownloadDesktopInstaller(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	inst, err := model.GetDesktopInstaller(r.Context(), s.db, id)
	if err != nil {
		if err == sql.ErrNoRows {
			s.writeError(w, http.StatusNotFound, "installer not found")
			return
		}
		log.Printf("get installer for download: %v", err)
		s.writeError(w, http.StatusInternalServerError, "fetch failed")
		return
	}

	dest, err := s.installerDownloadURL(r, inst, "")
	if err != nil {
		log.Printf("installer download URL: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to generate download URL")
		return
	}
	http.Redirect(w, r, dest, http.StatusFound)
}

// EmailDesktopInstallerInvite POST /api/desktop-installers/{id}/email
//
// Body: { recipient_email, recipient_name?, expiry_days? }
// Creates a one-time invite row + sends an email containing the install link.
// Admin-only.
func (s *Server) EmailDesktopInstallerInvite(w http.ResponseWriter, r *http.Request) {
	if s.cfg.SMTPHost == "" {
		s.writeError(w, http.StatusServiceUnavailable,
			"email is not configured on this server (set SMTP_HOST)")
		return
	}

	id := r.PathValue("id")
	inst, err := model.GetDesktopInstaller(r.Context(), s.db, id)
	if err != nil {
		if err == sql.ErrNoRows {
			s.writeError(w, http.StatusNotFound, "installer not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, "fetch failed")
		return
	}

	var req struct {
		RecipientEmail string `json:"recipient_email"`
		RecipientName  string `json:"recipient_name"`
		ExpiryDays     int    `json:"expiry_days"`
		InstitutionID  string `json:"institution_id"` // optional — when set, the invite shows up in the per-institution detail panel
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	req.RecipientEmail = strings.TrimSpace(req.RecipientEmail)
	if req.RecipientEmail == "" || !strings.Contains(req.RecipientEmail, "@") {
		s.writeError(w, http.StatusBadRequest, "valid recipient_email required")
		return
	}
	if req.ExpiryDays <= 0 || req.ExpiryDays > 90 {
		req.ExpiryDays = 7
	}
	req.InstitutionID = strings.TrimSpace(req.InstitutionID)
	if req.InstitutionID != "" {
		// Validate up front so a typo doesn't produce a 500 from the FK violation.
		if _, err := model.GetInstitutionByID(r.Context(), s.db, req.InstitutionID); err != nil {
			if err == sql.ErrNoRows {
				s.writeError(w, http.StatusBadRequest, "institution_id does not match a known institution")
				return
			}
			s.writeError(w, http.StatusInternalServerError, "institution lookup failed")
			return
		}
	}

	inv, err := model.CreateDesktopInstallerInvite(r.Context(), s.db, model.CreateDesktopInstallerInviteInput{
		InstallerID:    inst.ID,
		InstitutionID:  req.InstitutionID,
		RecipientEmail: req.RecipientEmail,
		RecipientName:  req.RecipientName,
		ExpiresAt:      time.Now().UTC().Add(time.Duration(req.ExpiryDays) * 24 * time.Hour),
		SentBy:         actorEmail(r),
	})
	if err != nil {
		log.Printf("create installer invite: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to create invite")
		return
	}

	productLabel := allowedInstallerProducts[inst.Product]
	platformLabel := allowedInstallerPlatforms[inst.Platform]
	installURL := strings.TrimRight(s.cfg.LandingBaseURL, "/") + "/install/" + inv.Token

	subject, body := email.DesktopInstallerInvite(inv.RecipientName, productLabel, platformLabel, installURL, inv.ExpiresAt)
	if err := s.mailer.Send(r.Context(), inv.RecipientEmail, subject, body); err != nil {
		log.Printf("send installer invite to %s: %v", inv.RecipientEmail, err)
		s.writeError(w, http.StatusInternalServerError, "failed to send email")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "desktop_installer.invite_sent", actorEmail(r),
		"desktop_installer", inst.ID, clientIP(r), map[string]any{
			"to": inv.RecipientEmail, "expiry_days": req.ExpiryDays,
		})

	s.writeJSON(w, http.StatusCreated, map[string]any{
		"status":      "sent",
		"invite_id":   inv.ID,
		"install_url": installURL,
		"expires_at":  inv.ExpiresAt,
	})
}

// ListDesktopInstallerInvites GET /api/desktop-installers/invites
//
// Returns recent invite history (default 100) for the admin dashboard.
func (s *Server) ListDesktopInstallerInvites(w http.ResponseWriter, r *http.Request) {
	out, err := model.ListDesktopInstallerInvites(r.Context(), s.db, 100)
	if err != nil {
		log.Printf("list installer invites: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to list invites")
		return
	}
	if out == nil {
		out = []model.DesktopInstallerInvite{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"invites": out})
}

// ListInstitutionInstallerInvites GET /api/institutions/{id}/installer-invites
//
// Returns recent invites tied to one institution. Used by the per-institution
// detail panel to show "browser/desktop install invitations" alongside the
// satellite list and (soon) the browser upload allowlist.
func (s *Server) ListInstitutionInstallerInvites(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		s.writeError(w, http.StatusBadRequest, "institution id required")
		return
	}
	if _, err := model.GetInstitutionByID(r.Context(), s.db, id); err != nil {
		if err == sql.ErrNoRows {
			s.writeError(w, http.StatusNotFound, "institution not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, "institution lookup failed")
		return
	}
	out, err := model.ListDesktopInstallerInvitesByInstitution(r.Context(), s.db, id, 100)
	if err != nil {
		log.Printf("list institution installer invites %s: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "failed to list invites")
		return
	}
	if out == nil {
		out = []model.DesktopInstallerInvite{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"invites": out})
}

// InstallLandingPage GET /install/{token}
//
// Public. Renders a minimal HTML page with platform-specific instructions
// and a download button that links to a fresh signed URL (with the pairing
// token embedded in the download filename for first-launch auto-pairing).
//
// Click metrics (count + first/last timestamps) are updated on every page load.
func (s *Server) InstallLandingPage(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	inv, inst, err := model.GetDesktopInstallerInviteByToken(r.Context(), s.db, token)
	if err != nil {
		if err == sql.ErrNoRows {
			renderInstallError(w, http.StatusNotFound, "Install link not found",
				"This install link is invalid or has been deleted. Please contact your administrator for a fresh link.")
			return
		}
		log.Printf("install landing lookup: %v", err)
		renderInstallError(w, http.StatusInternalServerError, "Internal error",
			"Something went wrong looking up this install link. Please try again.")
		return
	}
	if time.Now().UTC().After(inv.ExpiresAt) {
		renderInstallError(w, http.StatusGone, "Install link expired",
			"This install link has expired. Please contact your administrator for a fresh link.")
		return
	}

	if err := model.RecordDesktopInstallerInviteClick(r.Context(), s.db, inv.ID); err != nil {
		log.Printf("record install click: %v", err)
		// Non-fatal — still serve the page.
	}

	// Compute the download URL with pairing token embedded in download filename.
	dlFilename := installerDownloadFilename(inst, token)
	downloadURL, err := s.installerDownloadURL(r, inst, dlFilename)
	if err != nil {
		log.Printf("install landing download URL: %v", err)
		renderInstallError(w, http.StatusInternalServerError, "Download unavailable",
			"Could not generate a download link. Please try again in a moment.")
		return
	}

	renderInstallLanding(w, installLandingData{
		ProductLabel:  allowedInstallerProducts[inst.Product],
		PlatformLabel: allowedInstallerPlatforms[inst.Platform],
		Version:       inst.Version,
		SizeMB:        float64(inst.SizeBytes) / (1024 * 1024),
		SHA256:        inst.SHA256,
		Changelog:     inst.Changelog,
		DownloadURL:   downloadURL,
		DownloadName:  dlFilename,
		RecipientName: inv.RecipientName,
		Product:       inst.Product,
		Platform:      inst.Platform,
	})
}

// PairDesktopInstaller POST /api/install/pair
//
// Body: { pairing_token }
// Public. On success creates an API key scoped to this recipient and returns
// {api_key, server_url} for the desktop app to store. The pairing token is
// single-use — a second call returns 410.
func (s *Server) PairDesktopInstaller(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PairingToken string `json:"pairing_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PairingToken == "" {
		s.writeError(w, http.StatusBadRequest, "pairing_token required")
		return
	}

	// Generate a fresh API key. We use the same hash + prefix convention as
	// CreateAPIKey so the existing api_key middleware accepts these keys.
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to generate key")
		return
	}
	rawKey := "aegis_" + base64.RawURLEncoding.EncodeToString(buf)
	h := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(h[:])
	keyPrefix := rawKey[:min(14, len(rawKey))]

	// Look up the invite first so we can name the key meaningfully even before
	// we commit the claim. ClaimDesktopInstallerInvitePairing below is the
	// atomic "first writer wins" guard against double-use.
	apiKey, err := model.CreateAPIKey(r.Context(), s.db,
		"desktop-pairing-pending", keyHash, keyPrefix, "install-pair", nil)
	if err != nil {
		log.Printf("create api key for pairing: %v", err)
		s.writeError(w, http.StatusInternalServerError, "key creation failed")
		return
	}

	inv, ok, err := model.ClaimDesktopInstallerInvitePairing(r.Context(), s.db, req.PairingToken, apiKey.ID)
	if err != nil {
		log.Printf("claim pairing: %v", err)
		_ = model.DeleteAPIKey(r.Context(), s.db, apiKey.ID)
		s.writeError(w, http.StatusInternalServerError, "claim failed")
		return
	}
	if !ok {
		// Either the token is unknown, already paired, or expired.
		_ = model.DeleteAPIKey(r.Context(), s.db, apiKey.ID)
		s.writeError(w, http.StatusGone, "pairing token is invalid, already used, or expired")
		return
	}

	// Browser upload allowlist (chunk 2). If the invite is institution-
	// scoped and that institution has disabled desktop installer
	// pairing, deny here AFTER the claim (so the token still burns and
	// can't be retried elsewhere) and AFTER api-key creation (so we
	// can clean up the orphaned key). Institution-less invites bypass
	// the check — they're admin-issued and pre-date the allowlist.
	if !s.enforceUploadMethod(w, r, inv.InstitutionID, "desktop.installer-pair") {
		_ = model.DeleteAPIKey(r.Context(), s.db, apiKey.ID)
		return
	}

	// Now that the claim succeeded, rename the key so audit reads make sense.
	if err := s.renameAPIKey(r, apiKey.ID, inv); err != nil {
		log.Printf("rename paired api key: %v", err)
	}

	model.CreateAuditEntry(r.Context(), s.db, "desktop_installer.paired", inv.RecipientEmail,
		"desktop_installer", inv.InstallerID, clientIP(r), map[string]any{
			"recipient":   inv.RecipientEmail,
			"invite_id":   inv.ID,
			"api_key_id":  apiKey.ID,
		})

	s.writeJSON(w, http.StatusOK, map[string]any{
		"api_key":    rawKey,
		"server_url": s.cfg.LandingBaseURL,
	})
}

// renameAPIKey best-effort updates the freshly-created API key name from the
// placeholder "desktop-pairing-pending" to something the admin can recognise
// in the API keys tab.
func (s *Server) renameAPIKey(r *http.Request, apiKeyID string, inv *model.DesktopInstallerInvite) error {
	name := fmt.Sprintf("desktop:%s:%s", inv.Product, inv.RecipientEmail)
	_, err := s.db.ExecContext(r.Context(),
		`UPDATE api_keys SET name = $1, updated_at = now() WHERE id = $2`, name, apiKeyID)
	return err
}

// ----------------------------------------------------------------------------
// Helpers

// installerDownloadURL returns a URL the browser can follow to download the
// installer binary. For Storage-hosted installers it generates a signed URL
// valid for 1 hour. For external_url installers it returns the external URL
// directly. When forceFilename is non-empty, the signed URL is augmented with
// a response-content-disposition query parameter so the browser saves the
// file under that name (used to embed the pairing token in the filename).
func (s *Server) installerDownloadURL(r *http.Request, inst *model.DesktopInstaller, forceFilename string) (string, error) {
	if inst.ExternalURL != "" {
		return inst.ExternalURL, nil
	}
	signed, err := s.store.GenerateDownloadURL(r.Context(), inst.StorageKey, time.Hour)
	if err != nil {
		return "", err
	}
	if forceFilename == "" {
		return signed, nil
	}
	// response-content-disposition is honoured by GCS, S3, and Azure (via
	// rscd= in SAS). We append it via URL parsing so we don't double-encode
	// existing query parameters in the signed URL.
	u, err := url.Parse(signed)
	if err != nil {
		// Fall back to the un-renamed URL rather than failing the download.
		return signed, nil
	}
	q := u.Query()
	q.Set("response-content-disposition",
		fmt.Sprintf(`attachment; filename="%s"`, forceFilename))
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// installerDownloadFilename builds a filename that embeds the pairing token,
// so the desktop app can read its own argv[0] on first launch and pair without
// the user pasting anything. Example:
//   aegis-uploader-1.0.0-pair-abc123.dmg
func installerDownloadFilename(inst *model.DesktopInstaller, pairingToken string) string {
	// Find the extension. If the stored filename has none, fall back to .bin.
	base := inst.Filename
	ext := ""
	if dot := strings.LastIndex(base, "."); dot > 0 {
		ext = base[dot:]
		base = base[:dot]
	}
	if ext == "" {
		ext = ".bin"
	}
	// Trim long tokens to keep filenames manageable; the pairing token is the
	// FULL secret on the server side — we only need enough characters here for
	// the user to recognise their download.
	short := pairingToken
	if len(short) > 16 {
		short = short[:16]
	}
	return fmt.Sprintf("%s-pair-%s%s", base, short, ext)
}

// ----------------------------------------------------------------------------
// HTML rendering for the public landing page.

type installLandingData struct {
	ProductLabel  string
	PlatformLabel string
	Version       string
	SizeMB        float64
	SHA256        string
	Changelog     string
	DownloadURL   string
	DownloadName  string
	RecipientName string
	Product       string
	Platform      string
}

func renderInstallLanding(w http.ResponseWriter, d installLandingData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")

	greeting := "Hi"
	if d.RecipientName != "" {
		greeting = "Hi " + html.EscapeString(d.RecipientName)
	}

	fmt.Fprintf(w, `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Install %[1]s — AEGIS</title>
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>
  body { font: 16px/1.5 -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
         margin: 0; background: #f8fafc; color: #0f172a; }
  main { max-width: 640px; margin: 48px auto; padding: 32px;
         background: #fff; border-radius: 12px;
         box-shadow: 0 1px 3px rgba(0,0,0,.05), 0 1px 2px rgba(0,0,0,.04); }
  h1 { font-size: 24px; margin: 0 0 8px; }
  .meta { color: #475569; font-size: 14px; margin-bottom: 24px; }
  .download-btn { display: inline-block; padding: 14px 28px;
                  background: #0d9488; color: #fff; text-decoration: none;
                  border-radius: 8px; font-weight: 600; font-size: 16px; }
  .download-btn:hover { background: #0f766e; }
  .download-meta { color: #64748b; font-size: 13px; margin-top: 12px;
                   font-family: ui-monospace, SFMono-Regular, monospace;
                   word-break: break-all; }
  .section { margin-top: 32px; padding-top: 24px; border-top: 1px solid #e2e8f0; }
  .section h2 { font-size: 16px; margin: 0 0 12px; color: #334155; }
  .install-steps { padding-left: 20px; margin: 0; color: #334155; }
  .install-steps li { margin-bottom: 8px; }
  code { background: #f1f5f9; padding: 2px 6px; border-radius: 4px;
         font-family: ui-monospace, SFMono-Regular, monospace; font-size: 13px; }
  .changelog { background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 6px;
               padding: 12px 16px; white-space: pre-wrap; font-size: 14px; color: #334155; }
</style>
</head>
<body>
<main>
  <h1>%[2]s, here's your AEGIS installer.</h1>
  <p class="meta">%[1]s &middot; %[3]s &middot; version %[4]s</p>

  <p><a class="download-btn" href="%[5]s" download="%[6]s">Download installer</a></p>
  <p class="download-meta">
    File: %[6]s<br>
    Size: %[7].1f MB<br>
    SHA-256: %[8]s
  </p>
`,
		html.EscapeString(d.ProductLabel),
		greeting,
		html.EscapeString(d.PlatformLabel),
		html.EscapeString(d.Version),
		html.EscapeString(d.DownloadURL),
		html.EscapeString(d.DownloadName),
		d.SizeMB,
		html.EscapeString(d.SHA256),
	)

	fmt.Fprintf(w, `
  <section class="section">
    <h2>Installation</h2>
    <ol class="install-steps">
      %s
    </ol>
    <p style="color:#64748b; font-size:13px; margin-top:16px;">
      The first time you launch the app, it will pair with your AEGIS account
      automatically using a one-time token embedded in the download filename.
      Don't rename the file before installing.
    </p>
  </section>
`, installStepsFor(d.Platform, d.Product))

	if d.Changelog != "" {
		fmt.Fprintf(w, `
  <section class="section">
    <h2>What's new</h2>
    <div class="changelog">%s</div>
  </section>
`, html.EscapeString(d.Changelog))
	}

	fmt.Fprintf(w, `
  <section class="section" style="color:#64748b; font-size:13px;">
    Trouble installing? Contact your AEGIS administrator with the SHA-256 above
    and the time you tried to install.
  </section>
</main>
</body>
</html>`)
}

func installStepsFor(platform, product string) string {
	switch {
	case strings.HasPrefix(platform, "macos"):
		return `<li>Open the downloaded <code>.dmg</code> and drag the app to <code>Applications</code>.</li>
<li>Launch the app from <code>Applications</code>; macOS Gatekeeper may take a few seconds the first time.</li>
<li>The app will pair automatically. If prompted, allow it to access files.</li>`
	case platform == "windows-x64":
		if product == "dimse-bridge" {
			return `<li>Right-click the <code>.msi</code> and choose <strong>Install</strong>. Admin rights are required.</li>
<li>The installer will register a Windows Service named <code>AEGISDimseBridge</code>.</li>
<li>Configuration lives at <code>%ProgramData%\AEGIS\bridge.yml</code>.</li>`
		}
		return `<li>Run the downloaded installer. If Windows SmartScreen warns, click <strong>More info</strong> → <strong>Run anyway</strong>.</li>
<li>Launch <strong>AEGIS Uploader</strong> from the Start menu when installation finishes.</li>
<li>The app will pair automatically on first launch.</li>`
	case platform == "linux-deb":
		return `<li>Install with <code>sudo dpkg -i &lt;file&gt;.deb</code> (or <code>sudo apt install ./&lt;file&gt;.deb</code> to pull missing dependencies).</li>
<li>Launch the app from your application menu.</li>
<li>The app will pair automatically on first launch.</li>`
	case platform == "linux-appimage":
		return `<li>Mark the AppImage executable: <code>chmod +x &lt;file&gt;.AppImage</code>.</li>
<li>Run it directly: <code>./&lt;file&gt;.AppImage</code>.</li>
<li>The app will pair automatically on first launch.</li>`
	case platform == "linux-rpm":
		return `<li>Install with <code>sudo rpm -i &lt;file&gt;.rpm</code> (or <code>sudo dnf install &lt;file&gt;.rpm</code>).</li>
<li>Launch the app from your application menu.</li>
<li>The app will pair automatically on first launch.</li>`
	default:
		return `<li>Follow the platform-specific instructions provided with the download.</li>`
	}
}

func renderInstallError(w http.ResponseWriter, status int, title, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	fmt.Fprintf(w, `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><title>%[1]s — AEGIS</title>
<style>
  body { font: 16px/1.5 -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
         margin: 0; background: #f8fafc; color: #0f172a; }
  main { max-width: 560px; margin: 64px auto; padding: 32px;
         background: #fff; border-radius: 12px;
         box-shadow: 0 1px 3px rgba(0,0,0,.05); }
  h1 { font-size: 22px; margin: 0 0 12px; color: #9a3412; }
  p { color: #334155; }
</style></head><body><main>
<h1>%[1]s</h1><p>%[2]s</p>
</main></body></html>`,
		html.EscapeString(title), html.EscapeString(body))
}
