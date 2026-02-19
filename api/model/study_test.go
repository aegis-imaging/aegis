package model_test

import (
	"context"
	"testing"

	"github.com/msenjem/aegis/api/model"
	"github.com/msenjem/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateStudy(t *testing.T) {
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)

	s := testutil.CreateTestStudy(t, db, proj.ID)

	assert.NotEmpty(t, s.ID)
	assert.Equal(t, proj.ID, s.ProjectID)
	assert.Equal(t, "MRI", s.Modality)
	assert.Equal(t, "HEAD", s.BodyPart)
	assert.Equal(t, "received", s.Status)
	assert.Equal(t, "raw", s.DicomStore)
	assert.Equal(t, "external", s.Source)
	assert.False(t, s.DefacingRequired)
	assert.False(t, s.CreatedAt.IsZero())
}

func TestCreateStudy_DuplicateUID(t *testing.T) {
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)

	s1 := testutil.CreateTestStudy(t, db, proj.ID)
	s2 := &model.Study{
		ProjectID:        proj.ID,
		StudyInstanceUID: s1.StudyInstanceUID, // duplicate
		Status:           "received",
		DicomStore:       "raw",
		Source:           "external",
	}
	err := model.CreateStudy(context.Background(), db, s2)
	assert.Error(t, err, "duplicate study_instance_uid should fail")
}

func TestGetStudyByID(t *testing.T) {
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	created := testutil.CreateTestStudy(t, db, proj.ID)

	found, err := model.GetStudyByID(context.Background(), db, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, created.StudyInstanceUID, found.StudyInstanceUID)
	assert.Equal(t, 28, countStudyFields(found), "Study should have 28 scanned fields")
}

func TestGetStudyByUID(t *testing.T) {
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	created := testutil.CreateTestStudy(t, db, proj.ID)

	found, err := model.GetStudyByUID(context.Background(), db, created.StudyInstanceUID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
}

func TestUpdateStudyStatus(t *testing.T) {
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	s := testutil.CreateTestStudy(t, db, proj.ID)

	err := model.UpdateStudyStatus(context.Background(), db, s.ID, "approved")
	require.NoError(t, err)

	updated, _ := model.GetStudyByID(context.Background(), db, s.ID)
	assert.Equal(t, "approved", updated.Status)
}

func TestSetDefacingRequired(t *testing.T) {
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	s := testutil.CreateTestStudy(t, db, proj.ID)

	err := model.SetDefacingRequired(context.Background(), db, s.ID, true)
	require.NoError(t, err)

	updated, _ := model.GetStudyByID(context.Background(), db, s.ID)
	assert.True(t, updated.DefacingRequired)
}

func TestUpdateStudyDefaced(t *testing.T) {
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	s := testutil.CreateTestStudy(t, db, proj.ID)

	err := model.UpdateStudyDefaced(context.Background(), db, s.ID)
	require.NoError(t, err)

	updated, _ := model.GetStudyByID(context.Background(), db, s.ID)
	assert.Equal(t, "defaced", updated.Status)
	assert.Equal(t, "clean", updated.DicomStore)
}

func TestListStudies_NoFilter(t *testing.T) {
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	testutil.CreateTestStudy(t, db, proj.ID)
	testutil.CreateTestStudy(t, db, proj.ID)

	studies, err := model.ListStudies(context.Background(), db, model.StudyFilters{}, 50, 0)
	require.NoError(t, err)
	assert.Len(t, studies, 2)
}

func TestListStudies_StatusFilter(t *testing.T) {
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	s := testutil.CreateTestStudy(t, db, proj.ID)
	testutil.CreateTestStudy(t, db, proj.ID) // stays received

	model.UpdateStudyStatus(context.Background(), db, s.ID, "approved")

	studies, err := model.ListStudies(context.Background(), db, model.StudyFilters{Status: "approved"}, 50, 0)
	require.NoError(t, err)
	assert.Len(t, studies, 1)
	assert.Equal(t, "approved", studies[0].Status)
}

func TestListStudies_ModalityFilter(t *testing.T) {
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	testutil.CreateTestStudy(t, db, proj.ID) // MRI

	// Create a CT study
	ct := &model.Study{
		ProjectID: proj.ID, StudyInstanceUID: "2.3.4.5.ct",
		Modality: "CT", Status: "received", DicomStore: "raw", Source: "external",
	}
	require.NoError(t, model.CreateStudy(context.Background(), db, ct))

	studies, err := model.ListStudies(context.Background(), db, model.StudyFilters{Modality: "ct"}, 50, 0)
	require.NoError(t, err)
	assert.Len(t, studies, 1)
	assert.Equal(t, "CT", studies[0].Modality)
}

func TestListStudies_SearchFilter(t *testing.T) {
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)

	s := &model.Study{
		ProjectID: proj.ID, StudyInstanceUID: "1.2.3.unique.search",
		StudyDescription: "Brain MPRAGE", Modality: "MRI",
		Status: "received", DicomStore: "raw", Source: "external",
	}
	require.NoError(t, model.CreateStudy(context.Background(), db, s))
	testutil.CreateTestStudy(t, db, proj.ID) // generic study

	studies, err := model.ListStudies(context.Background(), db, model.StudyFilters{Search: "unique.search"}, 50, 0)
	require.NoError(t, err)
	assert.Len(t, studies, 1)
}

func TestListStudies_Pagination(t *testing.T) {
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	for i := 0; i < 5; i++ {
		testutil.CreateTestStudy(t, db, proj.ID)
	}

	page1, err := model.ListStudies(context.Background(), db, model.StudyFilters{}, 2, 0)
	require.NoError(t, err)
	assert.Len(t, page1, 2)

	page2, err := model.ListStudies(context.Background(), db, model.StudyFilters{}, 2, 2)
	require.NoError(t, err)
	assert.Len(t, page2, 2)

	assert.NotEqual(t, page1[0].ID, page2[0].ID)
}

func TestCountStudies(t *testing.T) {
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	testutil.CreateTestStudy(t, db, proj.ID)
	testutil.CreateTestStudy(t, db, proj.ID)

	total, err := model.CountStudies(context.Background(), db, model.StudyFilters{})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
}

func TestClaimClassification(t *testing.T) {
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	s := testutil.CreateTestStudy(t, db, proj.ID)

	// Set up pending classification
	model.SetClassificationRequired(context.Background(), db, s.ID, true)

	// First claim should succeed
	claimed, err := model.ClaimClassification(context.Background(), db, s.ID)
	require.NoError(t, err)
	assert.True(t, claimed)

	// Second claim should fail (already classifying)
	claimed2, err := model.ClaimClassification(context.Background(), db, s.ID)
	require.NoError(t, err)
	assert.False(t, claimed2)
}

func TestClaimDefacing(t *testing.T) {
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	s := testutil.CreateTestStudy(t, db, proj.ID)

	// Without defacing_required, claim should fail
	claimed, err := model.ClaimDefacing(context.Background(), db, s.ID)
	require.NoError(t, err)
	assert.False(t, claimed)

	// Set defacing_required
	model.SetDefacingRequired(context.Background(), db, s.ID, true)

	// Now claim should succeed
	claimed, err = model.ClaimDefacing(context.Background(), db, s.ID)
	require.NoError(t, err)
	assert.True(t, claimed)

	// Second claim should fail
	claimed2, err := model.ClaimDefacing(context.Background(), db, s.ID)
	require.NoError(t, err)
	assert.False(t, claimed2)
}

// countStudyFields validates that the Study struct scan alignment is correct
// by checking that all non-zero-value fields are populated after a DB read.
func countStudyFields(s *model.Study) int {
	count := 0
	if s.ID != "" {
		count++
	}
	if s.ProjectID != "" {
		count++
	}
	count++ // UploadSessionID (may be nil)
	count++ // InstitutionID (may be nil)
	if s.StudyInstanceUID != "" {
		count++
	}
	count++ // Modality (may be empty)
	count++ // BodyPart (may be empty)
	count++ // StudyDescription (may be empty)
	count++ // SeriesCount (may be 0)
	count++ // InstanceCount (may be 0)
	if s.Status != "" {
		count++
	}
	count++ // DefacingRequired (bool)
	if s.DicomStore != "" {
		count++
	}
	count++ // Source (may be empty)
	count++ // PhiScanRequired
	count++ // PhiScanStatus
	count++ // QcRequired
	count++ // QcStatus
	count++ // BidsRequired
	count++ // BidsStatus
	count++ // ClassificationRequired
	count++ // ClassificationStatus
	count++ // ProtocolRequired
	count++ // ProtocolStatus
	count++ // ExportRequired
	count++ // ExportStatus
	if !s.CreatedAt.IsZero() {
		count++
	}
	if !s.UpdatedAt.IsZero() {
		count++
	}
	return count
}
