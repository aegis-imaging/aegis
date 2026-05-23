package handler

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
)

// SetInstitutionClientCert PUT /api/institutions/{id}/client-cert
// Body: {"cert_pem": "..."} OR {"thumbprint": "<sha256 hex>"}
// (Whichever form the operator has handy.) Saves the cert identity to the
// institution so spoke-mTLS middleware can match incoming uploads.
func (s *Server) SetInstitutionClientCert(w http.ResponseWriter, r *http.Request) {
	instID := r.PathValue("id")

	var req struct {
		CertPEM    string `json:"cert_pem"`
		Thumbprint string `json:"thumbprint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	thumb := strings.ToLower(strings.TrimSpace(req.Thumbprint))
	thumb = strings.ReplaceAll(thumb, ":", "")
	subjectDN := ""

	if pemRaw := strings.TrimSpace(req.CertPEM); pemRaw != "" {
		block, _ := pem.Decode([]byte(pemRaw))
		if block == nil {
			s.writeError(w, http.StatusBadRequest, "cert_pem is not a valid PEM block")
			return
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, "cert_pem could not be parsed: "+err.Error())
			return
		}
		sum := sha256.Sum256(cert.Raw)
		thumb = hex.EncodeToString(sum[:])
		subjectDN = cert.Subject.String()
	}

	if thumb == "" {
		s.writeError(w, http.StatusBadRequest, "either cert_pem or thumbprint is required")
		return
	}

	if err := model.SetInstitutionClientCert(r.Context(), s.db, instID, thumb, subjectDN); err != nil {
		s.writeError(w, http.StatusInternalServerError, "save failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "institution.client_cert_set",
		actorEmail(r), "institution", instID, clientIP(r), map[string]any{
			"thumbprint":  thumb,
			"subject_dn":  subjectDN,
		})
	s.writeJSON(w, http.StatusOK, map[string]string{
		"institution_id": instID,
		"thumbprint":     thumb,
		"subject_dn":     subjectDN,
	})
}

// RevokeInstitutionClientCert DELETE /api/institutions/{id}/client-cert
func (s *Server) RevokeInstitutionClientCert(w http.ResponseWriter, r *http.Request) {
	instID := r.PathValue("id")
	if err := model.SetInstitutionClientCert(r.Context(), s.db, instID, "", ""); err != nil {
		s.writeError(w, http.StatusInternalServerError, "revoke failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "institution.client_cert_revoked",
		actorEmail(r), "institution", instID, clientIP(r), nil)
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}
