package handler_test

import (
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

func TestEditStudyComment_RecordsHistory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	c := &model.StudyComment{StudyID: study.ID, Author: "alice@test.com", Body: "initial"}
	require.NoError(t, model.CreateStudyComment(context.Background(), db, c))

	editReq := httptest.NewRequest(http.MethodPut, "/api/comments/"+c.ID, strings.NewReader(`{"body":"revised"}`))
	editReq.SetPathValue("id", c.ID)
	editRR := httptest.NewRecorder()
	srv.EditStudyComment(editRR, editReq)
	require.Equal(t, http.StatusOK, editRR.Code, editRR.Body.String())

	histReq := httptest.NewRequest(http.MethodGet, "/api/comments/"+c.ID+"/history", nil)
	histReq.SetPathValue("id", c.ID)
	histRR := httptest.NewRecorder()
	srv.ListCommentEditHistory(histRR, histReq)
	require.Equal(t, http.StatusOK, histRR.Code)

	var history []model.CommentEditHistory
	require.NoError(t, json.NewDecoder(histRR.Body).Decode(&history))
	require.Len(t, history, 1)
	assert.Equal(t, "initial", history[0].OldBody)
	assert.Equal(t, "revised", history[0].NewBody)
}

func TestEditStudyComment_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodPut, "/api/comments/00000000-0000-0000-0000-000000000000",
		strings.NewReader(`{"body":"hi"}`))
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	rr := httptest.NewRecorder()
	srv.EditStudyComment(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestEditStudyComment_EmptyBodyRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodPut, "/api/comments/x", strings.NewReader(`{"body":"   "}`))
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	rr := httptest.NewRecorder()
	srv.EditStudyComment(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
