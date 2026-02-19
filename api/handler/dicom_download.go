package handler

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/msenjem/aegis/api/model"
)

// ServeDicomDownload streams a zip archive of a study's DICOM files.
// Admin-protected, read-only (like BIDS download).
func (s *Server) ServeDicomDownload(w http.ResponseWriter, r *http.Request) {
	studyUID := r.PathValue("studyUID")
	study, err := model.GetStudyByUID(r.Context(), s.db, studyUID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}
	if study.Status != "approved" {
		s.writeError(w, http.StatusBadRequest, "only approved studies can be downloaded")
		return
	}

	s.streamDicomZip(w, r, study)

	model.CreateAuditEntry(r.Context(), s.db, "dicom_download.admin", actorEmail(r),
		"study", study.ID, clientIP(r), map[string]any{
			"study_uid": studyUID,
		})
}

// ServeDicomDownloadByToken streams a zip archive via a share token (public, no auth).
func (s *Server) ServeDicomDownloadByToken(w http.ResponseWriter, r *http.Request) {
	rawToken := r.PathValue("token")
	if rawToken == "" {
		s.writeError(w, http.StatusBadRequest, "missing token")
		return
	}

	h := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(h[:])

	share, err := model.GetExportShareByTokenHash(r.Context(), s.db, tokenHash)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "share not found")
		return
	}
	if share.RevokedAt != nil {
		s.writeError(w, http.StatusGone, "share has been revoked")
		return
	}
	if time.Now().After(share.ExpiresAt) {
		s.writeError(w, http.StatusGone, "share has expired")
		return
	}

	study, err := model.GetStudyByID(r.Context(), s.db, share.StudyID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "study not found")
		return
	}

	s.streamDicomZip(w, r, study)

	model.CreateExportDownload(r.Context(), s.db, share.ID, clientIP(r))
	model.CreateAuditEntry(r.Context(), s.db, "dicom_download.export", share.RecipientEmail,
		"export_share", share.ID, clientIP(r), map[string]any{
			"study_uid":  study.StudyInstanceUID,
			"file_count": study.InstanceCount,
		})
}

// streamDicomZip writes a zip archive of all DICOM files for a study to the response.
// Uses the storage interface (works with local, S3, and GCS backends).
func (s *Server) streamDicomZip(w http.ResponseWriter, r *http.Request, study *model.Study) {
	prefix := fmt.Sprintf("dicom/%s/%s", study.DicomStore, study.StudyInstanceUID)
	keys, err := s.store.List(r.Context(), prefix)
	if err != nil {
		log.Printf("dicom_download: list files for %s: %v", study.StudyInstanceUID, err)
		s.writeError(w, http.StatusInternalServerError, "failed to list study files")
		return
	}
	if len(keys) == 0 {
		s.writeError(w, http.StatusNotFound, "no DICOM files found")
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="%s_dicom.zip"`, study.StudyInstanceUID))

	zw := zip.NewWriter(w)
	defer zw.Close()

	for _, key := range keys {
		rc, err := s.store.Retrieve(r.Context(), key)
		if err != nil {
			log.Printf("dicom_download: retrieve %s: %v", key, err)
			continue
		}
		parts := strings.Split(key, "/")
		filename := parts[len(parts)-1]
		fw, err := zw.Create(filename)
		if err != nil {
			rc.Close()
			log.Printf("dicom_download: create zip entry %s: %v", filename, err)
			continue
		}
		io.Copy(fw, rc)
		rc.Close()
	}
}
