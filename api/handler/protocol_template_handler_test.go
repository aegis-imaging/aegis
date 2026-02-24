package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListProtocolTemplates_Empty(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/protocol-templates", nil)
	req.SetPathValue("projectID", proj.ID)
	rr := httptest.NewRecorder()

	srv.ListProtocolTemplates(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result []model.ProtocolTemplate
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Empty(t, result)
}

func TestCreateProtocolTemplate_Handler(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	body := map[string]any{
		"name":          "T1w MPRAGE Siemens Prisma",
		"manufacturer":  "SIEMENS",
		"model":         "MAGNETOM Prisma",
		"sequence_type": "T1w_MPRAGE",
		"rules":         json.RawMessage(`[]`),
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/protocol-templates", bytes.NewReader(b))
	req.SetPathValue("projectID", proj.ID)
	rr := httptest.NewRecorder()

	srv.CreateProtocolTemplate(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	var result model.ProtocolTemplate
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.NotEmpty(t, result.ID)
	assert.Equal(t, "T1w MPRAGE Siemens Prisma", result.Name)
	assert.Equal(t, "SIEMENS", result.Manufacturer)
	assert.Equal(t, "MAGNETOM Prisma", result.Model)
	assert.Equal(t, "T1w_MPRAGE", result.SequenceType)
	assert.Equal(t, proj.ID, result.ProjectID)
	assert.True(t, result.Enabled)
}

func TestCreateProtocolTemplate_MissingName(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	body := map[string]any{"manufacturer": "SIEMENS"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/protocol-templates", bytes.NewReader(b))
	req.SetPathValue("projectID", proj.ID)
	rr := httptest.NewRecorder()

	srv.CreateProtocolTemplate(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "name is required")
}

func TestCreateProtocolTemplate_ProjectNotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body := map[string]any{"name": "My Template"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/nonexistent/protocol-templates", bytes.NewReader(b))
	req.SetPathValue("projectID", "nonexistent-project-id")
	rr := httptest.NewRecorder()

	srv.CreateProtocolTemplate(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "project not found")
}

func TestListProtocolTemplates_ReturnsCreated(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Create two templates
	for _, name := range []string{"T1w Template", "FLAIR Template"} {
		body := map[string]any{"name": name, "sequence_type": "T1w_MPRAGE"}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/protocol-templates", bytes.NewReader(b))
		req.SetPathValue("projectID", proj.ID)
		rr := httptest.NewRecorder()
		srv.CreateProtocolTemplate(rr, req)
		require.Equal(t, http.StatusCreated, rr.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/protocol-templates", nil)
	req.SetPathValue("projectID", proj.ID)
	rr := httptest.NewRecorder()
	srv.ListProtocolTemplates(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result []model.ProtocolTemplate
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Len(t, result, 2)
}

func TestGetProtocolTemplate_Handler(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	tmpl := &model.ProtocolTemplate{
		ProjectID:    proj.ID,
		Name:         "CT Head",
		Manufacturer: "GE",
		SequenceType: "CT_head",
		Enabled:      true,
		Rules:        json.RawMessage(`[]`),
	}
	require.NoError(t, model.CreateProtocolTemplate(t.Context(), db, tmpl))

	req := httptest.NewRequest(http.MethodGet, "/api/protocol-templates/"+tmpl.ID, nil)
	req.SetPathValue("id", tmpl.ID)
	rr := httptest.NewRecorder()

	srv.GetProtocolTemplate(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result model.ProtocolTemplate
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, tmpl.ID, result.ID)
	assert.Equal(t, "CT Head", result.Name)
	assert.Equal(t, "GE", result.Manufacturer)
}

func TestGetProtocolTemplate_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/protocol-templates/missing", nil)
	req.SetPathValue("id", "missing-template-id")
	rr := httptest.NewRecorder()

	srv.GetProtocolTemplate(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "template not found")
}

func TestUpdateProtocolTemplate_Handler(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	tmpl := &model.ProtocolTemplate{
		ProjectID:    proj.ID,
		Name:         "Original Name",
		Manufacturer: "SIEMENS",
		SequenceType: "T1w_MPRAGE",
		Enabled:      true,
		Rules:        json.RawMessage(`[]`),
	}
	require.NoError(t, model.CreateProtocolTemplate(t.Context(), db, tmpl))

	update := map[string]any{
		"name":          "Updated Name",
		"manufacturer":  "PHILIPS",
		"model":         "Ingenia",
		"sequence_type": "FLAIR",
		"enabled":       false,
		"rules":         json.RawMessage(`[]`),
	}
	b, _ := json.Marshal(update)
	req := httptest.NewRequest(http.MethodPut, "/api/protocol-templates/"+tmpl.ID, bytes.NewReader(b))
	req.SetPathValue("id", tmpl.ID)
	rr := httptest.NewRecorder()

	srv.UpdateProtocolTemplate(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result model.ProtocolTemplate
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, "Updated Name", result.Name)
	assert.Equal(t, "PHILIPS", result.Manufacturer)
	assert.Equal(t, "Ingenia", result.Model)
	assert.Equal(t, "FLAIR", result.SequenceType)
	assert.False(t, result.Enabled)
}

func TestUpdateProtocolTemplate_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body := map[string]any{"name": "X"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/protocol-templates/missing", bytes.NewReader(b))
	req.SetPathValue("id", "no-such-template")
	rr := httptest.NewRecorder()

	srv.UpdateProtocolTemplate(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "template not found")
}

func TestDeleteProtocolTemplate_Handler(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	tmpl := &model.ProtocolTemplate{
		ProjectID: proj.ID,
		Name:      "To Delete",
		Enabled:   true,
		Rules:     json.RawMessage(`[]`),
	}
	require.NoError(t, model.CreateProtocolTemplate(t.Context(), db, tmpl))

	req := httptest.NewRequest(http.MethodDelete, "/api/protocol-templates/"+tmpl.ID, nil)
	req.SetPathValue("id", tmpl.ID)
	rr := httptest.NewRecorder()

	srv.DeleteProtocolTemplate(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)

	// Verify it's gone
	req2 := httptest.NewRequest(http.MethodGet, "/api/protocol-templates/"+tmpl.ID, nil)
	req2.SetPathValue("id", tmpl.ID)
	rr2 := httptest.NewRecorder()
	srv.GetProtocolTemplate(rr2, req2)
	assert.Equal(t, http.StatusNotFound, rr2.Code)
}

func TestDeleteProtocolTemplate_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodDelete, "/api/protocol-templates/missing", nil)
	req.SetPathValue("id", "no-such-template")
	rr := httptest.NewRecorder()

	srv.DeleteProtocolTemplate(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "template not found")
}

// ── Import handler tests ──────────────────────────────────────────────────────

func TestImportProtocolTemplates_ArrayFormat(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	payload := []map[string]any{
		{"name": "T1w MPRAGE", "manufacturer": "SIEMENS", "sequence_type": "T1w_MPRAGE", "rules": json.RawMessage(`[]`)},
		{"name": "FLAIR", "manufacturer": "PHILIPS", "sequence_type": "FLAIR", "rules": json.RawMessage(`[]`)},
	}
	b, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/protocol-templates/import", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("projectID", proj.ID)
	rr := httptest.NewRecorder()

	srv.ImportProtocolTemplates(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, float64(2), result["imported"])
	assert.Equal(t, float64(0), result["skipped"])
}

func TestImportProtocolTemplates_ExportFormat(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	payload := map[string]any{
		"project_id": proj.ID,
		"count":      1,
		"templates": []map[string]any{
			{"name": "DWI Protocol", "manufacturer": "GE", "sequence_type": "DWI", "rules": json.RawMessage(`[]`)},
		},
	}
	b, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/protocol-templates/import", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("projectID", proj.ID)
	rr := httptest.NewRecorder()

	srv.ImportProtocolTemplates(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, float64(1), result["imported"])
	assert.Equal(t, float64(0), result["skipped"])
}

func TestImportProtocolTemplates_SkipsDuplicates(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Pre-create a template with the same name.
	existing := &model.ProtocolTemplate{
		ProjectID: proj.ID, Name: "T1w Existing", Enabled: true, Rules: json.RawMessage(`[]`),
	}
	require.NoError(t, model.CreateProtocolTemplate(t.Context(), db, existing))

	payload := []map[string]any{
		{"name": "T1w Existing", "manufacturer": "SIEMENS", "rules": json.RawMessage(`[]`)}, // should be skipped
		{"name": "New Template", "manufacturer": "PHILIPS", "rules": json.RawMessage(`[]`)}, // should be imported
	}
	b, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/protocol-templates/import", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("projectID", proj.ID)
	rr := httptest.NewRecorder()

	srv.ImportProtocolTemplates(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, float64(1), result["imported"])
	assert.Equal(t, float64(1), result["skipped"])
}

func TestImportProtocolTemplates_InvalidJSON(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/protocol-templates/import",
		bytes.NewReader([]byte(`not valid json`)))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("projectID", proj.ID)
	rr := httptest.NewRecorder()

	srv.ImportProtocolTemplates(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid JSON")
}

func TestImportProtocolTemplates_EmptyArray(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/protocol-templates/import",
		bytes.NewReader([]byte(`[]`)))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("projectID", proj.ID)
	rr := httptest.NewRecorder()

	srv.ImportProtocolTemplates(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, float64(0), result["imported"])
	assert.Equal(t, float64(0), result["skipped"])
}

func TestImportProtocolTemplates_ProjectNotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	payload := []map[string]any{{"name": "Template"}}
	b, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/nonexistent/protocol-templates/import", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("projectID", "nonexistent-project-id")
	rr := httptest.NewRecorder()

	srv.ImportProtocolTemplates(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "project not found")
}
