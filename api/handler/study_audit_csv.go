package handler

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
)

// ExportStudyAuditCSV streams all audit entries for a specific study as a CSV file.
// GET /api/studies/{id}/audit.csv
func (s *Server) ExportStudyAuditCSV(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		s.writeError(w, http.StatusBadRequest, "missing study id")
		return
	}

	if _, _, ok := s.requireStudyReadAccessByID(w, r, id); !ok {
		return
	}

	entries, err := model.ListAuditEntriesForStudy(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list audit entries")
		return
	}

	filename := "audit-" + id + ".csv"
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"id", "created_at", "action", "actor", "resource_type", "resource_id", "ip_address", "detail"})

	for _, e := range entries {
		detail := ""
		if len(e.Detail) > 0 && string(e.Detail) != "null" {
			b, _ := json.Marshal(json.RawMessage(e.Detail))
			detail = string(b)
		}
		_ = cw.Write([]string{
			e.ID,
			e.CreatedAt.UTC().Format(time.RFC3339),
			e.Action,
			e.Actor,
			e.ResourceType,
			e.ResourceID,
			e.IPAddress,
			detail,
		})
	}

	cw.Flush()
}
