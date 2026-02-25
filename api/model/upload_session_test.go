package model_test

import (
	"context"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateUploadSession(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)

	s, err := model.CreateUploadSession(context.Background(), db,
		proj.ID, 5, "dicom/raw/test-prefix", "192.168.1.1", "uploader@example.com")
	require.NoError(t, err)
	assert.NotEmpty(t, s.ID)
	assert.Equal(t, proj.ID, s.ProjectID)
	assert.Equal(t, 5, s.FileCount)
	assert.Equal(t, "dicom/raw/test-prefix", s.StoragePrefix)
	assert.Equal(t, "192.168.1.1", s.UploaderIP)
	assert.Equal(t, "uploader@example.com", s.UploaderEmail)
	assert.Equal(t, "initiated", s.Status)
	assert.Nil(t, s.StudyInstanceUID)
	assert.Nil(t, s.ErrorMessage)
}

func TestGetUploadSession(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	created, err := model.CreateUploadSession(ctx, db, proj.ID, 3, "dicom/raw/get-test", "10.0.0.1", "")
	require.NoError(t, err)

	got, err := model.GetUploadSession(ctx, db, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, proj.ID, got.ProjectID)
	assert.Equal(t, 3, got.FileCount)
}

func TestUpdateUploadSessionStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	s, err := model.CreateUploadSession(ctx, db, proj.ID, 1, "dicom/raw/status-test", "", "")
	require.NoError(t, err)
	assert.Equal(t, "initiated", s.Status)

	require.NoError(t, model.UpdateUploadSessionStatus(ctx, db, s.ID, "uploaded"))

	got, err := model.GetUploadSession(ctx, db, s.ID)
	require.NoError(t, err)
	assert.Equal(t, "uploaded", got.Status)
}

func TestUpdateUploadSessionComplete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	s, err := model.CreateUploadSession(ctx, db, proj.ID, 10, "dicom/raw/complete-test", "", "")
	require.NoError(t, err)

	require.NoError(t, model.UpdateUploadSessionComplete(ctx, db, s.ID, "1.2.3.4.99", "MRI", "HEAD"))

	got, err := model.GetUploadSession(ctx, db, s.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", got.Status)
	require.NotNil(t, got.StudyInstanceUID)
	assert.Equal(t, "1.2.3.4.99", *got.StudyInstanceUID)
	require.NotNil(t, got.Modality)
	assert.Equal(t, "MRI", *got.Modality)
	require.NotNil(t, got.BodyPart)
	assert.Equal(t, "HEAD", *got.BodyPart)
	assert.Nil(t, got.ErrorMessage)
}

func TestUpdateUploadSessionFailed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	s, err := model.CreateUploadSession(ctx, db, proj.ID, 2, "dicom/raw/fail-test", "", "")
	require.NoError(t, err)

	require.NoError(t, model.UpdateUploadSessionFailed(ctx, db, s.ID, "checksum mismatch"))

	got, err := model.GetUploadSession(ctx, db, s.ID)
	require.NoError(t, err)
	assert.Equal(t, "failed", got.Status)
	require.NotNil(t, got.ErrorMessage)
	assert.Equal(t, "checksum mismatch", *got.ErrorMessage)
}
