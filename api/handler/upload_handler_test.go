package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── UploadInit ───────────────────────────────────────────────────────────────

func TestUploadInit_Success(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db) // reuses helper from dicom_download_handler_test.go

	body, _ := json.Marshal(map[string]any{"file_count": 2})
	req := httptest.NewRequest(http.MethodPost, "/api/upload/init", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	srv.UploadInit(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.NotEmpty(t, result["session_id"])
	urls, ok := result["upload_urls"].([]any)
	require.True(t, ok, "upload_urls should be a list")
	assert.Len(t, urls, 2)
	assert.NotEmpty(t, result["expires_at"])
}

func TestUploadInit_ZeroFileCount(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)

	body, _ := json.Marshal(map[string]any{"file_count": 0})
	req := httptest.NewRequest(http.MethodPost, "/api/upload/init", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	srv.UploadInit(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "file_count must be positive")
}

func TestUploadInit_NegativeFileCount(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)

	body, _ := json.Marshal(map[string]any{"file_count": -1})
	req := httptest.NewRequest(http.MethodPost, "/api/upload/init", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	srv.UploadInit(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "file_count must be positive")
}

func TestUploadInit_UnknownProject(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)

	body, _ := json.Marshal(map[string]any{
		"file_count":   1,
		"project_slug": "no-such-project-xyz",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/upload/init", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	srv.UploadInit(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "project not found")
}

func TestUploadInit_DefaultProject(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)

	// No project_slug → falls back to "default" which exists from migration 001.
	body, _ := json.Marshal(map[string]any{"file_count": 1})
	req := httptest.NewRequest(http.MethodPost, "/api/upload/init", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	srv.UploadInit(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.NotEmpty(t, result["session_id"])
	urls, _ := result["upload_urls"].([]any)
	assert.Len(t, urls, 1)
}

func TestUploadInit_WithUploaderEmail(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)

	body, _ := json.Marshal(map[string]any{
		"file_count":     1,
		"uploader_email": "radiologist@hospital.org",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/upload/init", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	srv.UploadInit(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.NotEmpty(t, result["session_id"])
}

func TestUploadInit_AssignsInstitutionByID(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)
	project := testutil.SeedProject(t, db)

	inst := createInstitution(t, db, "sender", "PACS_UPLOAD_ALPHA", true)
	linkInstitutionToProject(t, db, inst.ID, project.ID, "sender")

	body, _ := json.Marshal(map[string]any{
		"file_count":     1,
		"project_slug":   project.Slug,
		"institution_id": inst.ID,
		"uploader_email": "sender@hospital.org",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/upload/init", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	srv.UploadInit(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	sessionID, ok := result["session_id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, sessionID)

	session, err := model.GetUploadSession(context.Background(), db, sessionID)
	require.NoError(t, err)
	require.NotNil(t, session.InstitutionID)
	assert.Equal(t, inst.ID, *session.InstitutionID)
}

func TestUploadInit_RejectsInstitutionNotLinkedToProject(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)
	project := testutil.SeedProject(t, db)

	inst := createInstitution(t, db, "sender", "PACS_UPLOAD_BRAVO", true)
	otherProject, err := model.CreateProject(context.Background(), db, "Other Upload Project", "other-upload-project", "")
	require.NoError(t, err)
	linkInstitutionToProject(t, db, inst.ID, otherProject.ID, "sender")

	body, _ := json.Marshal(map[string]any{
		"file_count":     1,
		"project_slug":   project.Slug,
		"institution_id": inst.ID,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/upload/init", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	srv.UploadInit(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "institution is not linked to project as sender/admin")
}

// ─── UploadFile ───────────────────────────────────────────────────────────────

func TestUploadFile_Success(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)

	session, err := model.CreateUploadSession(context.Background(), db, proj.ID, 1, "", "127.0.0.1", "")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPut, "/api/upload/file/"+session.ID+"/0",
		strings.NewReader("DICM\x00fake-dicom-content"))
	req.SetPathValue("sessionID", session.ID)
	req.SetPathValue("index", "0")
	rr := httptest.NewRecorder()

	srv.UploadFile(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestUploadFile_SessionNotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)

	req := httptest.NewRequest(http.MethodPut, "/api/upload/file/no-such-session/0",
		strings.NewReader("data"))
	req.SetPathValue("sessionID", "no-such-session-uuid")
	req.SetPathValue("index", "0")
	rr := httptest.NewRecorder()

	srv.UploadFile(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "upload session not found")
}

func TestUploadFile_MissingPathValues(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)

	// Both path values empty (not set via SetPathValue).
	req := httptest.NewRequest(http.MethodPut, "/api/upload/file//", strings.NewReader("data"))
	rr := httptest.NewRecorder()

	srv.UploadFile(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "missing session ID or file index")
}

// ─── UploadComplete ───────────────────────────────────────────────────────────

func TestUploadComplete_SessionNotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)

	body, _ := json.Marshal(map[string]any{"session_id": "no-such-session"})
	req := httptest.NewRequest(http.MethodPost, "/api/upload/complete", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	srv.UploadComplete(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "upload session not found")
}

func TestUploadComplete_AlreadyProcessed(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)

	session, err := model.CreateUploadSession(context.Background(), db, proj.ID, 1, "", "", "")
	require.NoError(t, err)
	require.NoError(t, model.UpdateUploadSessionStatus(context.Background(), db, session.ID, "completed"))

	body, _ := json.Marshal(map[string]any{"session_id": session.ID})
	req := httptest.NewRequest(http.MethodPost, "/api/upload/complete", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	srv.UploadComplete(rr, req)

	assert.Equal(t, http.StatusConflict, rr.Code)
	assert.Contains(t, rr.Body.String(), "already processed")
}

func TestUploadComplete_NoFiles(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)

	session, err := model.CreateUploadSession(context.Background(), db, proj.ID, 1, "", "", "")
	require.NoError(t, err)

	body, _ := json.Marshal(map[string]any{"session_id": session.ID})
	req := httptest.NewRequest(http.MethodPost, "/api/upload/complete", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	srv.UploadComplete(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "no files found")
}

func TestUploadComplete_Success(t *testing.T) {
	db := testutil.TestDB(t)
	srv, store := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Create the session record directly.
	session, err := model.CreateUploadSession(context.Background(), db, proj.ID, 1, "", "127.0.0.1", "")
	require.NoError(t, err)

	// Write a staged file at the path UploadFile would create.
	stagingKey := "uploads/" + session.ID + "/0.dcm"
	require.NoError(t, store.Store(context.Background(), stagingKey, strings.NewReader("DICM\x00fake")))

	body, _ := json.Marshal(map[string]any{"session_id": session.ID})
	req := httptest.NewRequest(http.MethodPost, "/api/upload/complete", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	srv.UploadComplete(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, "completed", result["status"])
	assert.NotNil(t, result["study"])
}

// TestUploadComplete_FullFlow exercises the full upload flow end-to-end:
// UploadInit → UploadFile → UploadComplete.
func TestUploadComplete_FullFlow(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)

	// Step 1: init
	initBody, _ := json.Marshal(map[string]any{
		"file_count":     1,
		"uploader_email": "sender@hospital.org",
		"study_metadata": map[string]any{
			"modality":  "MR",
			"body_part": "HEAD",
		},
	})
	initReq := httptest.NewRequest(http.MethodPost, "/api/upload/init", bytes.NewReader(initBody))
	initRR := httptest.NewRecorder()
	srv.UploadInit(initRR, initReq)
	require.Equal(t, http.StatusOK, initRR.Code)

	var initResult map[string]any
	require.NoError(t, json.NewDecoder(initRR.Body).Decode(&initResult))
	sessionID := initResult["session_id"].(string)
	require.NotEmpty(t, sessionID)

	// Step 2: upload one file
	fileReq := httptest.NewRequest(http.MethodPut, "/api/upload/file/"+sessionID+"/0",
		strings.NewReader("DICM\x00fake-dicom-content"))
	fileReq.SetPathValue("sessionID", sessionID)
	fileReq.SetPathValue("index", "0")
	fileRR := httptest.NewRecorder()
	srv.UploadFile(fileRR, fileReq)
	require.Equal(t, http.StatusOK, fileRR.Code)

	// Step 3: complete
	completeBody, _ := json.Marshal(map[string]any{"session_id": sessionID})
	completeReq := httptest.NewRequest(http.MethodPost, "/api/upload/complete", bytes.NewReader(completeBody))
	completeRR := httptest.NewRecorder()
	srv.UploadComplete(completeRR, completeReq)

	require.Equal(t, http.StatusOK, completeRR.Code)
	var completeResult map[string]any
	require.NoError(t, json.NewDecoder(completeRR.Body).Decode(&completeResult))
	assert.Equal(t, "completed", completeResult["status"])
	assert.NotNil(t, completeResult["study"])
}
