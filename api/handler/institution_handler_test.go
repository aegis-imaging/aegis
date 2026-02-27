package handler_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedInstitution(t *testing.T, db *sql.DB) *model.Institution {
	t.Helper()
	ts := time.Now().UnixNano()
	inst := &model.Institution{
		Name:     fmt.Sprintf("Institution %d", ts),
		Slug:     fmt.Sprintf("institution-%d", ts),
		Type:     "sender",
		AETitle:  "SEED_AE",
		IPRanges: "10.0.0.0/24",
		Enabled:  true,
	}
	require.NoError(t, model.CreateInstitution(context.Background(), db, inst))
	return inst
}

func TestCreateInstitution_NormalizesNetworkIdentity(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]any{
		"name":             "Site Alpha",
		"institution_type": "sender",
		"ae_title":         "  pacs_alpha  ",
		"ip_ranges":        "10.0.0.1, 10.0.0.0/24, 10.0.0.1",
	})
	req := httptest.NewRequest("POST", "/api/institutions", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.CreateInstitution(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)

	var inst model.Institution
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&inst))
	assert.Equal(t, "PACS_ALPHA", inst.AETitle)
	assert.Equal(t, "10.0.0.1,10.0.0.0/24", inst.IPRanges)
}

func TestCreateInstitution_RejectsInvalidIPRanges(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]any{
		"name":             "Bad Network Site",
		"institution_type": "sender",
		"ip_ranges":        "10.0.0.0/24, not-a-cidr",
	})
	req := httptest.NewRequest("POST", "/api/institutions", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.CreateInstitution(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestUpdateInstitution_NormalizesNetworkIdentity(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	inst := seedInstitution(t, db)

	body, _ := json.Marshal(map[string]any{
		"name":             "Updated Sender Site",
		"slug":             "",
		"institution_type": "both",
		"ae_title":         "  dimse_sender  ",
		"ip_ranges":        "10.1.0.0/16,10.1.1.1,10.1.1.1",
		"enabled":          true,
	})
	req := httptest.NewRequest("PUT", "/api/institutions/"+inst.ID, bytes.NewReader(body))
	req.SetPathValue("id", inst.ID)
	rr := httptest.NewRecorder()
	srv.UpdateInstitution(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var updated model.Institution
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&updated))
	assert.Equal(t, "updated-sender-site", updated.Slug)
	assert.Equal(t, "DIMSE_SENDER", updated.AETitle)
	assert.Equal(t, "10.1.0.0/16,10.1.1.1", updated.IPRanges)
}

func TestUpdateInstitution_RejectsMissingName(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	inst := seedInstitution(t, db)

	body, _ := json.Marshal(map[string]any{
		"name":             "",
		"institution_type": "sender",
		"enabled":          true,
	})
	req := httptest.NewRequest("PUT", "/api/institutions/"+inst.ID, bytes.NewReader(body))
	req.SetPathValue("id", inst.ID)
	rr := httptest.NewRecorder()
	srv.UpdateInstitution(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestUpdateInstitution_RejectsInvalidType(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	inst := seedInstitution(t, db)

	body, _ := json.Marshal(map[string]any{
		"name":             "Site Invalid Type",
		"institution_type": "invalid",
		"enabled":          true,
	})
	req := httptest.NewRequest("PUT", "/api/institutions/"+inst.ID, bytes.NewReader(body))
	req.SetPathValue("id", inst.ID)
	rr := httptest.NewRecorder()
	srv.UpdateInstitution(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestListInstitutions_ResearcherRequiresProjectScope(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	researcher := testutil.CreateTestAdminUser(t, db, "inst-researcher@test.com", "researcher")

	req := httptest.NewRequest("GET", "/api/institutions", nil)
	req = withResearcherUser(req, researcher.ID, researcher.Email)
	rr := httptest.NewRecorder()

	srv.ListInstitutions(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "project_id is required for researcher queries")
}

func TestListInstitutions_ResearcherWithProjectScopeAllowed(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	inst := seedInstitution(t, db)
	researcher := testutil.CreateTestAdminUser(t, db, "inst-researcher-ok@test.com", "researcher")

	require.NoError(t, model.AddInstitutionToProject(context.Background(), db, &model.InstitutionProject{
		InstitutionID: inst.ID,
		ProjectID:     proj.ID,
		Role:          "sender",
	}))
	require.NoError(t, model.CreateProjectMember(context.Background(), db, &model.ProjectMember{
		ProjectID:   proj.ID,
		AdminUserID: researcher.ID,
		Role:        "coordinator",
	}))

	req := httptest.NewRequest("GET", "/api/institutions?project_id="+proj.ID, nil)
	req = withResearcherUser(req, researcher.ID, researcher.Email)
	rr := httptest.NewRecorder()

	srv.ListInstitutions(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var out []model.Institution
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&out))
	require.Len(t, out, 1)
	assert.Equal(t, inst.ID, out[0].ID)
}
