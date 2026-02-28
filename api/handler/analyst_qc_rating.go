package handler

import (
	"encoding/json"
	"net/http"

	"github.com/aegis-imaging/aegis/api/middleware"
	"github.com/aegis-imaging/aegis/api/model"
)

// analystQCRatingBody is the JSON body for submitting a QC rating.
type analystQCRatingBody struct {
	RatingQuality   int16  `json:"rating_quality"`
	RatingMotion    int16  `json:"rating_motion"`
	RatingSNR       *int16 `json:"rating_snr"`
	RatingCoverage  *int16 `json:"rating_coverage"`
	RatingArtifacts *int16 `json:"rating_artifacts"`
	RatingOverall   int16  `json:"rating_overall"`
	Comments        string `json:"comments"`
	ReviewType      string `json:"review_type"`
}

// ListQCRatings returns all QC ratings for a study.
// GET /api/studies/{id}/qc-ratings
func (s *Server) ListQCRatings(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")
	if studyID == "" {
		s.writeError(w, http.StatusBadRequest, "missing study ID")
		return
	}

	ratings, err := model.GetQCRatingsByStudy(r.Context(), s.db, studyID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query QC ratings")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"ratings": ratings,
		"total":   len(ratings),
	})
}

// SubmitQCRating creates or updates a QC rating for a study.
// POST /api/studies/{id}/qc-ratings
func (s *Server) SubmitQCRating(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")
	if studyID == "" {
		s.writeError(w, http.StatusBadRequest, "missing study ID")
		return
	}

	var body analystQCRatingBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if body.ReviewType == "" {
		body.ReviewType = "initial"
	}

	user := middleware.UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	rating := &model.AnalystQCRating{
		StudyID:         studyID,
		AnalystID:       user.ID,
		RatingQuality:   body.RatingQuality,
		RatingMotion:    body.RatingMotion,
		RatingSNR:       body.RatingSNR,
		RatingCoverage:  body.RatingCoverage,
		RatingArtifacts: body.RatingArtifacts,
		RatingOverall:   body.RatingOverall,
		Comments:        body.Comments,
		ReviewType:      body.ReviewType,
	}

	if err := model.UpsertAnalystQCRating(r.Context(), s.db, rating); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to save QC rating")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "qc_rating.submitted", actorEmail(r),
		"study", studyID, clientIP(r), map[string]any{
			"rating_id":      rating.ID,
			"review_type":    body.ReviewType,
			"rating_overall": body.RatingOverall,
		})

	s.writeJSON(w, http.StatusOK, rating)
}

// UpdateQCRating updates an existing QC rating.
// PUT /api/qc-ratings/{id}
func (s *Server) UpdateQCRating(w http.ResponseWriter, r *http.Request) {
	ratingID := r.PathValue("id")
	if ratingID == "" {
		s.writeError(w, http.StatusBadRequest, "missing rating ID")
		return
	}

	existing, err := model.GetQCRatingByID(r.Context(), s.db, ratingID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "QC rating not found")
		return
	}

	// Only the analyst who created the rating or an admin can update it.
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if user.ID != existing.AnalystID && user.Role != "admin" {
		s.writeError(w, http.StatusForbidden, "can only update your own ratings")
		return
	}

	var body analystQCRatingBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	existing.RatingQuality = body.RatingQuality
	existing.RatingMotion = body.RatingMotion
	existing.RatingSNR = body.RatingSNR
	existing.RatingCoverage = body.RatingCoverage
	existing.RatingArtifacts = body.RatingArtifacts
	existing.RatingOverall = body.RatingOverall
	existing.Comments = body.Comments

	// Re-upsert with the same study/analyst/review_type to update.
	if err := model.UpsertAnalystQCRating(r.Context(), s.db, existing); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to update QC rating")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "qc_rating.updated", actorEmail(r),
		"study", existing.StudyID, clientIP(r), map[string]any{
			"rating_id":      existing.ID,
			"rating_overall": body.RatingOverall,
		})

	s.writeJSON(w, http.StatusOK, existing)
}

// DeleteQCRating removes a QC rating.
// DELETE /api/qc-ratings/{id}
func (s *Server) DeleteQCRating(w http.ResponseWriter, r *http.Request) {
	ratingID := r.PathValue("id")
	if ratingID == "" {
		s.writeError(w, http.StatusBadRequest, "missing rating ID")
		return
	}

	existing, err := model.GetQCRatingByID(r.Context(), s.db, ratingID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "QC rating not found")
		return
	}

	user := middleware.UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if user.ID != existing.AnalystID && user.Role != "admin" {
		s.writeError(w, http.StatusForbidden, "can only delete your own ratings")
		return
	}

	if err := model.DeleteAnalystQCRating(r.Context(), s.db, ratingID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to delete QC rating")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "qc_rating.deleted", actorEmail(r),
		"study", existing.StudyID, clientIP(r), map[string]any{
			"rating_id": ratingID,
		})

	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
