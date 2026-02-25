package model_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeProtocolTemplate(projectID, name, manufacturer, model_, seqType string) *model.ProtocolTemplate {
	return &model.ProtocolTemplate{
		ProjectID:    projectID,
		Name:         name,
		Manufacturer: manufacturer,
		Model:        model_,
		SequenceType: seqType,
		Enabled:      true,
		Rules:        json.RawMessage(`[]`),
	}
}

func TestCreateProtocolTemplate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)

	tmpl := makeProtocolTemplate(proj.ID, "T1w MPRAGE", "SIEMENS", "MAGNETOM Prisma", "T1w_MPRAGE")
	require.NoError(t, model.CreateProtocolTemplate(context.Background(), db, tmpl))
	assert.NotEmpty(t, tmpl.ID)
	assert.False(t, tmpl.CreatedAt.IsZero())
}

func TestCreateProtocolTemplate_NilRulesDefaultsToEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)

	tmpl := &model.ProtocolTemplate{
		ProjectID:    proj.ID,
		Name:         "nil-rules-template",
		Manufacturer: "GE",
		SequenceType: "T2w",
		Enabled:      true,
		Rules:        nil, // should default to []
	}
	require.NoError(t, model.CreateProtocolTemplate(context.Background(), db, tmpl))
	assert.Equal(t, json.RawMessage("[]"), tmpl.Rules)
}

func TestGetProtocolTemplateByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	tmpl := makeProtocolTemplate(proj.ID, "FLAIR Template", "PHILIPS", "Ingenia", "FLAIR")
	require.NoError(t, model.CreateProtocolTemplate(ctx, db, tmpl))

	got, err := model.GetProtocolTemplateByID(ctx, db, tmpl.ID)
	require.NoError(t, err)
	assert.Equal(t, tmpl.ID, got.ID)
	assert.Equal(t, "FLAIR Template", got.Name)
	assert.Equal(t, "PHILIPS", got.Manufacturer)
	assert.Equal(t, "FLAIR", got.SequenceType)
}

func TestGetProtocolTemplateByID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)

	_, err := model.GetProtocolTemplateByID(context.Background(), db, "00000000-0000-0000-0000-000000000000")
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestListProtocolTemplatesByProject(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	otherProj := testutil.CreateTestProject(t, db, "proto-other-project")
	ctx := context.Background()

	t1 := makeProtocolTemplate(proj.ID, "Alpha Template", "SIEMENS", "", "T1w")
	t2 := makeProtocolTemplate(proj.ID, "Beta Template", "GE", "", "DWI")
	tOther := makeProtocolTemplate(otherProj.ID, "Other Template", "PHILIPS", "", "FLAIR")
	require.NoError(t, model.CreateProtocolTemplate(ctx, db, t1))
	require.NoError(t, model.CreateProtocolTemplate(ctx, db, t2))
	require.NoError(t, model.CreateProtocolTemplate(ctx, db, tOther))

	templates, err := model.ListProtocolTemplatesByProject(ctx, db, proj.ID)
	require.NoError(t, err)
	require.Len(t, templates, 2)
	// Ordered by name: Alpha before Beta.
	assert.Equal(t, "Alpha Template", templates[0].Name)
	assert.Equal(t, "Beta Template", templates[1].Name)
}

func TestUpdateProtocolTemplate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	tmpl := makeProtocolTemplate(proj.ID, "Update Me", "SIEMENS", "Prisma", "T1w")
	require.NoError(t, model.CreateProtocolTemplate(ctx, db, tmpl))

	tmpl.Name = "Updated Name"
	tmpl.Manufacturer = "GE"
	tmpl.Enabled = false
	tmpl.Rules = json.RawMessage(`[{"tag":"TR","target":2000,"tolerance":5}]`)
	require.NoError(t, model.UpdateProtocolTemplate(ctx, db, tmpl))

	got, err := model.GetProtocolTemplateByID(ctx, db, tmpl.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", got.Name)
	assert.Equal(t, "GE", got.Manufacturer)
	assert.False(t, got.Enabled)
}

func TestDeleteProtocolTemplate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	tmpl := makeProtocolTemplate(proj.ID, "Delete Me", "SIEMENS", "", "T2w")
	require.NoError(t, model.CreateProtocolTemplate(ctx, db, tmpl))
	require.NoError(t, model.DeleteProtocolTemplate(ctx, db, tmpl.ID))

	_, err := model.GetProtocolTemplateByID(ctx, db, tmpl.ID)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestFindMatchingTemplates_ExactMatch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	tmpl := makeProtocolTemplate(proj.ID, "Exact Match Template", "SIEMENS", "MAGNETOM Prisma", "T1w")
	require.NoError(t, model.CreateProtocolTemplate(ctx, db, tmpl))

	matches, err := model.FindMatchingTemplates(ctx, db, proj.ID, "SIEMENS", "MAGNETOM Prisma", "")
	require.NoError(t, err)
	var ids []string
	for _, m := range matches {
		ids = append(ids, m.ID)
	}
	assert.Contains(t, ids, tmpl.ID)
}

func TestFindMatchingTemplates_WildcardManufacturer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	// Empty manufacturer acts as wildcard.
	tmpl := makeProtocolTemplate(proj.ID, "Wildcard Mfr Template", "", "Any Model", "DWI")
	require.NoError(t, model.CreateProtocolTemplate(ctx, db, tmpl))

	matches, err := model.FindMatchingTemplates(ctx, db, proj.ID, "PHILIPS", "Any Model", "")
	require.NoError(t, err)
	var ids []string
	for _, m := range matches {
		ids = append(ids, m.ID)
	}
	assert.Contains(t, ids, tmpl.ID)
}

func TestFindMatchingTemplates_NoMatch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	tmpl := makeProtocolTemplate(proj.ID, "No Match Template", "SIEMENS", "Skyra", "FLAIR")
	require.NoError(t, model.CreateProtocolTemplate(ctx, db, tmpl))

	// Different manufacturer — should not match (explicit SIEMENS template vs GE query).
	matches, err := model.FindMatchingTemplates(ctx, db, proj.ID, "GE", "Discovery", "")
	require.NoError(t, err)
	for _, m := range matches {
		assert.NotEqual(t, tmpl.ID, m.ID)
	}
}

func TestFindMatchingTemplates_DisabledNotReturned(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	tmpl := makeProtocolTemplate(proj.ID, "Disabled Template", "SIEMENS", "", "T1w")
	tmpl.Enabled = false
	require.NoError(t, model.CreateProtocolTemplate(ctx, db, tmpl))

	matches, err := model.FindMatchingTemplates(ctx, db, proj.ID, "SIEMENS", "", "")
	require.NoError(t, err)
	for _, m := range matches {
		assert.NotEqual(t, tmpl.ID, m.ID, "disabled template should not appear in results")
	}
}
