package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStudyWriteAuthz_NonMemberDenied(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	inst := testutil.CreateTestInstitution(t, db, "p0-22-non-member")
	study := testutil.CreateTestStudy(t, db, proj.ID)
	_, err := db.ExecContext(context.Background(), `UPDATE studies SET institution_id = $1, qc_required = true, qc_status = 'pass', status = 'expired' WHERE id = $2`, inst.ID, study.ID)
	require.NoError(t, err)

	researcher := testutil.CreateTestAdminUser(t, db, "p0-22-non-member@test.com", "researcher")

	tests := []struct {
		name     string
		handler  func(http.ResponseWriter, *http.Request)
		method   string
		path     string
		body     string
		expected int
	}{
		{name: "note", handler: srv.AddStudyNote, method: "POST", path: "/api/studies/" + study.ID + "/notes", body: `{"note":"nm"}`, expected: http.StatusNotFound},
		{name: "flag", handler: srv.PatchStudyFlag, method: "PATCH", path: "/api/studies/" + study.ID + "/flag", body: `{"flagged":true}`, expected: http.StatusNotFound},
		{name: "reset", handler: srv.ResetPipelineStep, method: "POST", path: "/api/studies/" + study.ID + "/reset-pipeline-step", body: `{"step":"qc"}`, expected: http.StatusNotFound},
		{name: "approve", handler: srv.ApproveStudy, method: "POST", path: "/api/studies/" + study.ID + "/approve", expected: http.StatusNotFound},
		{name: "reject", handler: srv.RejectStudy, method: "POST", path: "/api/studies/" + study.ID + "/reject", expected: http.StatusNotFound},
		{name: "reactivate", handler: srv.ReactivateStudy, method: "POST", path: "/api/studies/" + study.ID + "/reactivate", expected: http.StatusNotFound},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var body *bytes.Buffer
			if tc.body != "" {
				body = bytes.NewBufferString(tc.body)
			} else {
				body = bytes.NewBuffer(nil)
			}
			req := httptest.NewRequest(tc.method, tc.path, body)
			req = withResearcherUser(req, researcher.ID, researcher.Email)
			req.Header.Set("Content-Type", "application/json")
			req.SetPathValue("id", study.ID)
			rr := httptest.NewRecorder()
			tc.handler(rr, req)
			assert.Equal(t, tc.expected, rr.Code)
		})
	}
}

func TestStudyWriteAuthz_SiteViewerDenied(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	inst := testutil.CreateTestInstitution(t, db, "p0-22-site-viewer")
	study := testutil.CreateTestStudy(t, db, proj.ID)
	_, err := db.ExecContext(context.Background(), `UPDATE studies SET institution_id = $1, qc_required = true, qc_status = 'pass', status = 'expired' WHERE id = $2`, inst.ID, study.ID)
	require.NoError(t, err)

	researcher := testutil.CreateTestAdminUser(t, db, "p0-22-site-viewer@test.com", "researcher")
	err = model.CreateProjectMember(context.Background(), db, &model.ProjectMember{
		ProjectID:     proj.ID,
		AdminUserID:   researcher.ID,
		Role:          "site_viewer",
		InstitutionID: &inst.ID,
	})
	require.NoError(t, err)

	tests := []struct {
		name     string
		handler  func(http.ResponseWriter, *http.Request)
		method   string
		path     string
		body     string
		expected int
	}{
		{name: "note", handler: srv.AddStudyNote, method: "POST", path: "/api/studies/" + study.ID + "/notes", body: `{"note":"sv"}`, expected: http.StatusForbidden},
		{name: "flag", handler: srv.PatchStudyFlag, method: "PATCH", path: "/api/studies/" + study.ID + "/flag", body: `{"flagged":true}`, expected: http.StatusForbidden},
		{name: "reset", handler: srv.ResetPipelineStep, method: "POST", path: "/api/studies/" + study.ID + "/reset-pipeline-step", body: `{"step":"qc"}`, expected: http.StatusForbidden},
		{name: "approve", handler: srv.ApproveStudy, method: "POST", path: "/api/studies/" + study.ID + "/approve", expected: http.StatusForbidden},
		{name: "reject", handler: srv.RejectStudy, method: "POST", path: "/api/studies/" + study.ID + "/reject", expected: http.StatusForbidden},
		{name: "reactivate", handler: srv.ReactivateStudy, method: "POST", path: "/api/studies/" + study.ID + "/reactivate", expected: http.StatusForbidden},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var body *bytes.Buffer
			if tc.body != "" {
				body = bytes.NewBufferString(tc.body)
			} else {
				body = bytes.NewBuffer(nil)
			}
			req := httptest.NewRequest(tc.method, tc.path, body)
			req = withResearcherUser(req, researcher.ID, researcher.Email)
			req.Header.Set("Content-Type", "application/json")
			req.SetPathValue("id", study.ID)
			rr := httptest.NewRecorder()
			tc.handler(rr, req)
			assert.Equal(t, tc.expected, rr.Code)
		})
	}
}

func TestStudyWriteAuthz_PositiveRoleMatrix(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	inst := testutil.CreateTestInstitution(t, db, "p0-22-positive")

	t.Run("owner approve allowed", func(t *testing.T) {
		study := testutil.CreateTestStudy(t, db, proj.ID)
		owner := testutil.CreateTestAdminUser(t, db, "p0-22-owner@test.com", "researcher")
		err := model.CreateProjectMember(context.Background(), db, &model.ProjectMember{ProjectID: proj.ID, AdminUserID: owner.ID, Role: "owner"})
		require.NoError(t, err)
		req := httptest.NewRequest("POST", "/api/studies/"+study.ID+"/approve", nil)
		req = withResearcherUser(req, owner.ID, owner.Email)
		req.SetPathValue("id", study.ID)
		rr := httptest.NewRecorder()
		srv.ApproveStudy(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("coordinator reject allowed", func(t *testing.T) {
		study := testutil.CreateTestStudy(t, db, proj.ID)
		coordinator := testutil.CreateTestAdminUser(t, db, "p0-22-coordinator@test.com", "researcher")
		err := model.CreateProjectMember(context.Background(), db, &model.ProjectMember{ProjectID: proj.ID, AdminUserID: coordinator.ID, Role: "coordinator"})
		require.NoError(t, err)
		req := httptest.NewRequest("POST", "/api/studies/"+study.ID+"/reject", nil)
		req = withResearcherUser(req, coordinator.ID, coordinator.Email)
		req.SetPathValue("id", study.ID)
		rr := httptest.NewRecorder()
		srv.RejectStudy(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("reviewer approve allowed", func(t *testing.T) {
		study := testutil.CreateTestStudy(t, db, proj.ID)
		reviewer := testutil.CreateTestAdminUser(t, db, "p0-22-reviewer@test.com", "researcher")
		err := model.CreateProjectMember(context.Background(), db, &model.ProjectMember{ProjectID: proj.ID, AdminUserID: reviewer.ID, Role: "reviewer"})
		require.NoError(t, err)
		req := httptest.NewRequest("POST", "/api/studies/"+study.ID+"/approve", nil)
		req = withResearcherUser(req, reviewer.ID, reviewer.Email)
		req.SetPathValue("id", study.ID)
		rr := httptest.NewRecorder()
		srv.ApproveStudy(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("site coordinator study mutation allowed", func(t *testing.T) {
		study := testutil.CreateTestStudy(t, db, proj.ID)
		_, err := db.ExecContext(context.Background(), `UPDATE studies SET institution_id = $1, qc_required = true, qc_status = 'pass' WHERE id = $2`, inst.ID, study.ID)
		require.NoError(t, err)
		siteCoordinator := testutil.CreateTestAdminUser(t, db, "p0-22-site-coordinator@test.com", "researcher")
		err = model.CreateProjectMember(context.Background(), db, &model.ProjectMember{ProjectID: proj.ID, AdminUserID: siteCoordinator.ID, Role: "site_coordinator", InstitutionID: &inst.ID})
		require.NoError(t, err)

		noteReq := httptest.NewRequest("POST", "/api/studies/"+study.ID+"/notes", bytes.NewBufferString(`{"note":"ok"}`))
		noteReq = withResearcherUser(noteReq, siteCoordinator.ID, siteCoordinator.Email)
		noteReq.Header.Set("Content-Type", "application/json")
		noteReq.SetPathValue("id", study.ID)
		noteRR := httptest.NewRecorder()
		srv.AddStudyNote(noteRR, noteReq)
		assert.Equal(t, http.StatusOK, noteRR.Code)

		flagReq := httptest.NewRequest("PATCH", "/api/studies/"+study.ID+"/flag", bytes.NewBufferString(`{"flagged":true}`))
		flagReq = withResearcherUser(flagReq, siteCoordinator.ID, siteCoordinator.Email)
		flagReq.Header.Set("Content-Type", "application/json")
		flagReq.SetPathValue("id", study.ID)
		flagRR := httptest.NewRecorder()
		srv.PatchStudyFlag(flagRR, flagReq)
		assert.Equal(t, http.StatusOK, flagRR.Code)

		resetReq := httptest.NewRequest("POST", "/api/studies/"+study.ID+"/reset-pipeline-step", bytes.NewBufferString(`{"step":"qc"}`))
		resetReq = withResearcherUser(resetReq, siteCoordinator.ID, siteCoordinator.Email)
		resetReq.Header.Set("Content-Type", "application/json")
		resetReq.SetPathValue("id", study.ID)
		resetRR := httptest.NewRecorder()
		srv.ResetPipelineStep(resetRR, resetReq)
		assert.Equal(t, http.StatusOK, resetRR.Code)
	})
}

func TestStudyWriteAuthz_AdminGlobalAccessAcrossProject(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest("POST", "/api/studies/"+study.ID+"/approve", nil)
	req = withAdminUser(req, "p0-22-admin", "p0-22-admin@test.com")
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.ApproveStudy(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "approved", resp["status"])
}
