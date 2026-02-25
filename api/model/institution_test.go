package model_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeInstitution(ts int64, suffix, instType string) *model.Institution {
	return &model.Institution{
		Name:    fmt.Sprintf("Inst %s %d", suffix, ts),
		Slug:    fmt.Sprintf("inst-%s-%d", suffix, ts),
		Type:    instType,
		Enabled: true,
	}
}

func TestCreateInstitution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ts := time.Now().UnixNano()

	inst := makeInstitution(ts, "create", "sender")
	inst.ContactEmail = "create@example.com"
	inst.IPRanges = "10.0.0.0/8"
	inst.AETitle = "CREATE_AE"

	require.NoError(t, model.CreateInstitution(context.Background(), db, inst))
	assert.NotEmpty(t, inst.ID)
	assert.False(t, inst.CreatedAt.IsZero())
}

func TestGetInstitutionByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ts := time.Now().UnixNano()
	ctx := context.Background()

	inst := makeInstitution(ts, "get-by-id", "receiver")
	inst.ContactName = "Dr. Smith"
	require.NoError(t, model.CreateInstitution(ctx, db, inst))

	got, err := model.GetInstitutionByID(ctx, db, inst.ID)
	require.NoError(t, err)
	assert.Equal(t, inst.ID, got.ID)
	assert.Equal(t, inst.Name, got.Name)
	assert.Equal(t, "Dr. Smith", got.ContactName)
	assert.Equal(t, "receiver", got.Type)
}

func TestGetInstitutionBySlug(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ts := time.Now().UnixNano()
	ctx := context.Background()

	inst := makeInstitution(ts, "get-by-slug", "both")
	require.NoError(t, model.CreateInstitution(ctx, db, inst))

	// Slug lookup is case-insensitive.
	got, err := model.GetInstitutionBySlug(ctx, db, inst.Slug)
	require.NoError(t, err)
	assert.Equal(t, inst.ID, got.ID)
}

func TestListInstitutions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ts := time.Now().UnixNano()
	ctx := context.Background()

	i1 := makeInstitution(ts, "list-a", "sender")
	i2 := makeInstitution(ts, "list-b", "receiver")
	require.NoError(t, model.CreateInstitution(ctx, db, i1))
	require.NoError(t, model.CreateInstitution(ctx, db, i2))

	all, err := model.ListInstitutions(ctx, db)
	require.NoError(t, err)
	var ids []string
	for _, i := range all {
		ids = append(ids, i.ID)
	}
	assert.Contains(t, ids, i1.ID)
	assert.Contains(t, ids, i2.ID)
}

func TestUpdateInstitution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ts := time.Now().UnixNano()
	ctx := context.Background()

	inst := makeInstitution(ts, "update", "sender")
	require.NoError(t, model.CreateInstitution(ctx, db, inst))

	inst.ContactEmail = "updated@example.com"
	inst.Type = "both"
	inst.Enabled = false
	require.NoError(t, model.UpdateInstitution(ctx, db, inst))

	got, err := model.GetInstitutionByID(ctx, db, inst.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated@example.com", got.ContactEmail)
	assert.Equal(t, "both", got.Type)
	assert.False(t, got.Enabled)
}

func TestDeleteInstitution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ts := time.Now().UnixNano()
	ctx := context.Background()

	inst := makeInstitution(ts, "delete", "sender")
	require.NoError(t, model.CreateInstitution(ctx, db, inst))
	require.NoError(t, model.DeleteInstitution(ctx, db, inst.ID))

	_, err := model.GetInstitutionByID(ctx, db, inst.ID)
	assert.Error(t, err)
}

func TestAddRemoveInstitutionToProject(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ts := time.Now().UnixNano()
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	inst := makeInstitution(ts, "link", "sender")
	require.NoError(t, model.CreateInstitution(ctx, db, inst))

	// Add link.
	link := &model.InstitutionProject{
		InstitutionID: inst.ID,
		ProjectID:     proj.ID,
		Role:          "sender",
	}
	require.NoError(t, model.AddInstitutionToProject(ctx, db, link))

	projs, err := model.ListProjectsForInstitution(ctx, db, inst.ID)
	require.NoError(t, err)
	require.Len(t, projs, 1)
	assert.Equal(t, proj.ID, projs[0].ProjectID)
	assert.Equal(t, "sender", projs[0].Role)

	// Remove link.
	require.NoError(t, model.RemoveInstitutionFromProject(ctx, db, inst.ID, proj.ID))

	projs2, err := model.ListProjectsForInstitution(ctx, db, inst.ID)
	require.NoError(t, err)
	assert.Empty(t, projs2)
}

func TestAddInstitutionToProject_UpdateRole(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ts := time.Now().UnixNano()
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	inst := makeInstitution(ts, "update-role", "both")
	require.NoError(t, model.CreateInstitution(ctx, db, inst))

	// Add with sender role.
	require.NoError(t, model.AddInstitutionToProject(ctx, db, &model.InstitutionProject{
		InstitutionID: inst.ID, ProjectID: proj.ID, Role: "sender",
	}))

	// Upsert to admin role.
	require.NoError(t, model.AddInstitutionToProject(ctx, db, &model.InstitutionProject{
		InstitutionID: inst.ID, ProjectID: proj.ID, Role: "admin",
	}))

	projs, err := model.ListProjectsForInstitution(ctx, db, inst.ID)
	require.NoError(t, err)
	require.Len(t, projs, 1)
	assert.Equal(t, "admin", projs[0].Role)
}

func TestListInstitutionsForProject(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ts := time.Now().UnixNano()
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	inst := makeInstitution(ts, "list-for-proj", "sender")
	require.NoError(t, model.CreateInstitution(ctx, db, inst))
	require.NoError(t, model.AddInstitutionToProject(ctx, db, &model.InstitutionProject{
		InstitutionID: inst.ID, ProjectID: proj.ID, Role: "sender",
	}))

	links, err := model.ListInstitutionsForProject(ctx, db, proj.ID)
	require.NoError(t, err)
	var instIDs []string
	for _, l := range links {
		instIDs = append(instIDs, l.InstitutionID)
	}
	assert.Contains(t, instIDs, inst.ID)
}

func TestGetInstitutionStats(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ts := time.Now().UnixNano()
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	inst := makeInstitution(ts, "stats", "sender")
	require.NoError(t, model.CreateInstitution(ctx, db, inst))

	// Create a study and link it to the institution.
	study := testutil.CreateTestStudy(t, db, proj.ID)
	_, err := db.ExecContext(ctx,
		`UPDATE studies SET institution_id = $1 WHERE id = $2`, inst.ID, study.ID)
	require.NoError(t, err)

	stats, err := model.GetInstitutionStats(ctx, db, inst.ID)
	require.NoError(t, err)
	assert.Equal(t, inst.ID, stats.InstitutionID)
	assert.Equal(t, 1, stats.TotalStudies)
	assert.NotNil(t, stats.LastStudyAt)
	assert.Equal(t, 1, stats.ByStatus["received"])
	assert.Equal(t, 1, stats.ByModality["MRI"])
}

func TestGetInstitutionStats_NoStudies(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ts := time.Now().UnixNano()

	inst := makeInstitution(ts, "stats-empty", "sender")
	require.NoError(t, model.CreateInstitution(context.Background(), db, inst))

	stats, err := model.GetInstitutionStats(context.Background(), db, inst.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, stats.TotalStudies)
	assert.Nil(t, stats.LastStudyAt)
}

func TestInstitutionCanSendToProject(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ts := time.Now().UnixNano()
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	inst := makeInstitution(ts, "can-send", "sender")
	require.NoError(t, model.CreateInstitution(ctx, db, inst))

	// Not linked yet → cannot send.
	ok, err := model.InstitutionCanSendToProject(ctx, db, inst.ID, proj.ID)
	require.NoError(t, err)
	assert.False(t, ok)

	// Link with sender role.
	require.NoError(t, model.AddInstitutionToProject(ctx, db, &model.InstitutionProject{
		InstitutionID: inst.ID, ProjectID: proj.ID, Role: "sender",
	}))

	ok, err = model.InstitutionCanSendToProject(ctx, db, inst.ID, proj.ID)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestInstitutionCanSendToProject_ReceiverRoleBlocked(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ts := time.Now().UnixNano()
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	// Institution type "receiver" cannot send.
	inst := makeInstitution(ts, "recv-blocked", "receiver")
	require.NoError(t, model.CreateInstitution(ctx, db, inst))
	require.NoError(t, model.AddInstitutionToProject(ctx, db, &model.InstitutionProject{
		InstitutionID: inst.ID, ProjectID: proj.ID, Role: "receiver",
	}))

	ok, err := model.InstitutionCanSendToProject(ctx, db, inst.ID, proj.ID)
	require.NoError(t, err)
	assert.False(t, ok)
}
