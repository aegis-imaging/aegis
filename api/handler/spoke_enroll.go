package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
)

// EnrollSpoke POST /api/spokes/enroll
//
// PUBLIC endpoint — no admin auth required. The enrollment token itself IS
// the credential. Each token redeems exactly once.
//
// Body: {"token": "<raw token>", "csr_pem": "<PEM CSR>", "site_id": "..."}
// Response: {"client_cert_pem", "ca_pem", "thumbprint", "subject_dn", "not_after"}
//
// On success we (a) sign the CSR with the AEGIS spoke-issuer CA, (b) record
// the resulting cert thumbprint on the institution the token belongs to so
// the mTLS middleware can later match incoming uploads, and (c) audit it.
func (s *Server) EnrollSpoke(w http.ResponseWriter, r *http.Request) {
	if s.spokeCA == nil {
		s.writeError(w, http.StatusServiceUnavailable, "spoke enrollment is disabled on this server")
		return
	}

	var req struct {
		Token   string `json:"token"`
		CSRPEM  string `json:"csr_pem"`
		SiteID  string `json:"site_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Token = strings.TrimSpace(req.Token)
	req.CSRPEM = strings.TrimSpace(req.CSRPEM)
	if req.Token == "" || req.CSRPEM == "" {
		s.writeError(w, http.StatusBadRequest, "token and csr_pem are required")
		return
	}

	token, err := model.RedeemSpokeEnrollmentToken(r.Context(), s.db, req.Token)
	if err != nil {
		if errors.Is(err, model.ErrEnrollmentTokenInvalid) {
			s.writeError(w, http.StatusUnauthorized, "enrollment token invalid or expired")
			return
		}
		s.writeError(w, http.StatusInternalServerError, "lookup failed")
		return
	}

	inst, err := model.GetInstitutionByID(r.Context(), s.db, token.InstitutionID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "institution lookup failed")
		return
	}

	fallbackCN := strings.TrimSpace(req.SiteID)
	if fallbackCN == "" {
		fallbackCN = inst.Slug
	}

	signed, err := s.spokeCA.Sign(req.CSRPEM, fallbackCN)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "CSR signing failed: "+err.Error())
		return
	}

	if err := model.SetInstitutionClientCert(r.Context(), s.db, inst.ID, signed.Thumbprint, signed.SubjectDN); err != nil {
		s.writeError(w, http.StatusInternalServerError, "could not record enrolled cert")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "spoke.enrolled",
		"system", "institution", inst.ID, clientIP(r), map[string]any{
			"token_id":   token.ID,
			"site_id":    fallbackCN,
			"thumbprint": signed.Thumbprint,
			"subject_dn": signed.SubjectDN,
			"not_after":  signed.NotAfter,
			"serial":     signed.SerialHex,
		})

	s.writeJSON(w, http.StatusOK, map[string]any{
		"client_cert_pem": signed.ClientCertPEM,
		"ca_pem":          signed.CAPEM,
		"thumbprint":      signed.Thumbprint,
		"subject_dn":      signed.SubjectDN,
		"not_after":       signed.NotAfter,
		"institution_id":  inst.ID,
		"institution_slug": inst.Slug,
	})
}
