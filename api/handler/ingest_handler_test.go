package handler_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/msenjem/aegis/api/model"
	"github.com/msenjem/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createInstitution(t *testing.T, db *sql.DB, instType, aeTitle string, enabled bool) *model.Institution {
	return createInstitutionWithIPRanges(t, db, instType, aeTitle, "", enabled)
}

func createInstitutionWithIPRanges(t *testing.T, db *sql.DB, instType, aeTitle, ipRanges string, enabled bool) *model.Institution {
	t.Helper()
	ts := time.Now().UnixNano()
	inst := &model.Institution{
		Name:     fmt.Sprintf("Institution %d", ts),
		Slug:     fmt.Sprintf("institution-%d", ts),
		Type:     instType,
		AETitle:  aeTitle,
		IPRanges: ipRanges,
		Enabled:  enabled,
	}
	require.NoError(t, model.CreateInstitution(context.Background(), db, inst))
	return inst
}

func linkInstitutionToProject(t *testing.T, db *sql.DB, institutionID, projectID, role string) {
	t.Helper()
	require.NoError(t, model.AddInstitutionToProject(context.Background(), db, &model.InstitutionProject{
		InstitutionID: institutionID,
		ProjectID:     projectID,
		Role:          role,
	}))
}

func newIngestPayload(studyUID string) map[string]any {
	return map[string]any{
		"project_slug": "default",
		"study_metadata": map[string]any{
			"study_instance_uid": studyUID,
			"modality":           "MR",
			"body_part":          "HEAD",
			"study_description":  "BRAIN MRI",
			"series_count":       1,
			"instance_count":     2,
		},
	}
}

func TestInternalIngest_AssignsInstitutionByID(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	project := testutil.SeedProject(t, db)

	inst := createInstitution(t, db, "sender", "PACS_ALPHA", true)
	linkInstitutionToProject(t, db, inst.ID, project.ID, "sender")

	payload := newIngestPayload(fmt.Sprintf("1.2.840.%d", time.Now().UnixNano()))
	payload["institution_id"] = inst.ID

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/ingest", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.InternalIngest(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)

	var resp struct {
		Study model.Study `json:"study"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.NotNil(t, resp.Study.InstitutionID)
	assert.Equal(t, inst.ID, *resp.Study.InstitutionID)
}

func TestInternalIngest_AssignsInstitutionBySlug(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	project := testutil.SeedProject(t, db)

	inst := createInstitution(t, db, "sender", "PACS_ALPHA", true)
	linkInstitutionToProject(t, db, inst.ID, project.ID, "sender")

	payload := newIngestPayload(fmt.Sprintf("1.2.840.%d", time.Now().UnixNano()))
	payload["institution_slug"] = strings.ToUpper(inst.Slug)

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/ingest", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.InternalIngest(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)

	var resp struct {
		Study model.Study `json:"study"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.NotNil(t, resp.Study.InstitutionID)
	assert.Equal(t, inst.ID, *resp.Study.InstitutionID)
}

func TestInternalIngest_AssignsInstitutionByAETitle(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	project := testutil.SeedProject(t, db)

	inst := createInstitution(t, db, "both", "PACS_BRAVO", true)
	linkInstitutionToProject(t, db, inst.ID, project.ID, "admin")

	payload := newIngestPayload(fmt.Sprintf("1.2.840.%d", time.Now().UnixNano()))
	payload["institution_ae_title"] = "pacs_bravo"

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/ingest", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.InternalIngest(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)

	var resp struct {
		Study model.Study `json:"study"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.NotNil(t, resp.Study.InstitutionID)
	assert.Equal(t, inst.ID, *resp.Study.InstitutionID)
}

func TestInternalIngest_RejectsAmbiguousInstitutionAETitle(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	project := testutil.SeedProject(t, db)

	instA := createInstitution(t, db, "sender", "PACS_DUP", true)
	instB := createInstitution(t, db, "sender", "PACS_DUP", true)
	linkInstitutionToProject(t, db, instA.ID, project.ID, "sender")
	linkInstitutionToProject(t, db, instB.ID, project.ID, "sender")

	payload := newIngestPayload(fmt.Sprintf("1.2.840.%d", time.Now().UnixNano()))
	payload["institution_ae_title"] = "pacs_dup"

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/ingest", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.InternalIngest(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "matched multiple enabled institutions")
}

func TestInternalIngest_RejectsInstitutionIDAndSlugConflict(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	project := testutil.SeedProject(t, db)

	inst := createInstitution(t, db, "sender", "PACS_CHARLIE", true)
	linkInstitutionToProject(t, db, inst.ID, project.ID, "sender")

	payload := newIngestPayload(fmt.Sprintf("1.2.840.%d", time.Now().UnixNano()))
	payload["institution_id"] = inst.ID
	payload["institution_slug"] = inst.Slug

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/ingest", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.InternalIngest(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "only one")
}

func TestInternalIngest_RejectsInstitutionNotLinkedToProject(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	_ = testutil.SeedProject(t, db)

	inst := createInstitution(t, db, "sender", "PACS_CHARLIE", true)

	payload := newIngestPayload(fmt.Sprintf("1.2.840.%d", time.Now().UnixNano()))
	payload["institution_id"] = inst.ID

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/ingest", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.InternalIngest(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestInternalIngest_RejectsInstitutionSlugAETitleMismatch(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	project := testutil.SeedProject(t, db)

	inst := createInstitution(t, db, "sender", "PACS_DELTA", true)
	linkInstitutionToProject(t, db, inst.ID, project.ID, "sender")

	payload := newIngestPayload(fmt.Sprintf("1.2.840.%d", time.Now().UnixNano()))
	payload["institution_slug"] = inst.Slug
	payload["institution_ae_title"] = "PACS_OTHER"

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/ingest", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.InternalIngest(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestInternalIngest_RejectsInstitutionIDAETitleMismatch(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	project := testutil.SeedProject(t, db)

	inst := createInstitution(t, db, "sender", "PACS_DELTA", true)
	linkInstitutionToProject(t, db, inst.ID, project.ID, "sender")

	payload := newIngestPayload(fmt.Sprintf("1.2.840.%d", time.Now().UnixNano()))
	payload["institution_id"] = inst.ID
	payload["institution_ae_title"] = "PACS_OTHER"

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/ingest", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.InternalIngest(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestInternalIngest_RejectsReceiverOnlyInstitution(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	project := testutil.SeedProject(t, db)

	inst := createInstitution(t, db, "receiver", "PACS_ECHO", true)
	linkInstitutionToProject(t, db, inst.ID, project.ID, "sender")

	payload := newIngestPayload(fmt.Sprintf("1.2.840.%d", time.Now().UnixNano()))
	payload["institution_id"] = inst.ID

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/ingest", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.InternalIngest(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestInternalIngest_AutoAssignsInstitutionBySourceIP(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	project := testutil.SeedProject(t, db)

	inst := createInstitutionWithIPRanges(t, db, "sender", "PACS_FOXTROT", "10.10.0.0/16", true)
	linkInstitutionToProject(t, db, inst.ID, project.ID, "sender")

	payload := newIngestPayload(fmt.Sprintf("1.2.840.%d", time.Now().UnixNano()))
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/ingest", bytes.NewReader(body))
	req.Header.Set("X-Forwarded-For", "10.10.12.30")
	rr := httptest.NewRecorder()
	srv.InternalIngest(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)

	var resp struct {
		Study model.Study `json:"study"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.NotNil(t, resp.Study.InstitutionID)
	assert.Equal(t, inst.ID, *resp.Study.InstitutionID)
}

func TestInternalIngest_AutoAssignsMostSpecificIPRange(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	project := testutil.SeedProject(t, db)

	broad := createInstitutionWithIPRanges(t, db, "sender", "PACS_GOLF", "10.0.0.0/8", true)
	narrow := createInstitutionWithIPRanges(t, db, "sender", "PACS_HOTEL", "10.11.0.0/16", true)
	linkInstitutionToProject(t, db, broad.ID, project.ID, "sender")
	linkInstitutionToProject(t, db, narrow.ID, project.ID, "sender")

	payload := newIngestPayload(fmt.Sprintf("1.2.840.%d", time.Now().UnixNano()))
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/ingest", bytes.NewReader(body))
	req.Header.Set("X-Forwarded-For", "10.11.22.33")
	rr := httptest.NewRecorder()
	srv.InternalIngest(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)

	var resp struct {
		Study model.Study `json:"study"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.NotNil(t, resp.Study.InstitutionID)
	assert.Equal(t, narrow.ID, *resp.Study.InstitutionID)
}

func TestInternalIngest_DoesNotFailWhenNoIPMatch(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	project := testutil.SeedProject(t, db)

	inst := createInstitutionWithIPRanges(t, db, "sender", "PACS_INDIA", "192.168.0.0/16", true)
	linkInstitutionToProject(t, db, inst.ID, project.ID, "sender")

	payload := newIngestPayload(fmt.Sprintf("1.2.840.%d", time.Now().UnixNano()))
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/ingest", bytes.NewReader(body))
	req.Header.Set("X-Forwarded-For", "10.55.1.9")
	rr := httptest.NewRecorder()
	srv.InternalIngest(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)

	var resp struct {
		Study model.Study `json:"study"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Nil(t, resp.Study.InstitutionID)
}

func TestInternalIngest_DoesNotAssignWhenIPMatchIsAmbiguous(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	project := testutil.SeedProject(t, db)

	instA := createInstitutionWithIPRanges(t, db, "sender", "PACS_JULIET", "10.77.0.0/16", true)
	instB := createInstitutionWithIPRanges(t, db, "sender", "PACS_KILO", "10.77.0.0/16", true)
	linkInstitutionToProject(t, db, instA.ID, project.ID, "sender")
	linkInstitutionToProject(t, db, instB.ID, project.ID, "sender")

	payload := newIngestPayload(fmt.Sprintf("1.2.840.%d", time.Now().UnixNano()))
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/ingest", bytes.NewReader(body))
	req.Header.Set("X-Forwarded-For", "10.77.42.9")
	rr := httptest.NewRecorder()
	srv.InternalIngest(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)

	var resp struct {
		Study model.Study `json:"study"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Nil(t, resp.Study.InstitutionID)
}
