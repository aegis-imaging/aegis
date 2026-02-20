package importer_test

import (
	"context"
	"testing"

	"github.com/msenjem/aegis/api/importer"
	"github.com/msenjem/aegis/api/model"
	"github.com/msenjem/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_DryRun_ResolvesInstitutionBySlug(t *testing.T) {
	db := testutil.TestDB(t)
	ctx := context.Background()

	project, err := model.GetProjectBySlug(ctx, db, "default")
	require.NoError(t, err)

	inst := &model.Institution{
		Name:    "Hospital Bravo",
		Slug:    "hospital-bravo",
		Type:    "sender",
		Enabled: true,
	}
	require.NoError(t, model.CreateInstitution(ctx, db, inst))
	require.NoError(t, model.AddInstitutionToProject(ctx, db, &model.InstitutionProject{
		InstitutionID: inst.ID,
		ProjectID:     project.ID,
		Role:          "sender",
	}))

	result, err := importer.Run(ctx, db, nil, importer.Options{
		Dir:             t.TempDir(),
		DryRun:          true,
		InstitutionSlug: "HOSPITAL-BRAVO",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 0, result.FilesScanned)
	assert.Equal(t, 0, result.StudiesCreated)
}

func TestRun_DryRun_RejectsUnknownInstitutionSlug(t *testing.T) {
	db := testutil.TestDB(t)

	_, err := importer.Run(context.Background(), db, nil, importer.Options{
		Dir:             t.TempDir(),
		DryRun:          true,
		InstitutionSlug: "missing-inst",
	})
	require.Error(t, err)
	assert.True(t, importer.IsValidationError(err))
	assert.Contains(t, err.Error(), "not found")
}

func TestRun_DryRun_ResolvesInstitutionByAETitle(t *testing.T) {
	db := testutil.TestDB(t)
	ctx := context.Background()

	project, err := model.GetProjectBySlug(ctx, db, "default")
	require.NoError(t, err)

	inst := &model.Institution{
		Name:    "Hospital Charlie",
		Slug:    "hospital-charlie",
		Type:    "sender",
		AETitle: "PACS_CHARLIE",
		Enabled: true,
	}
	require.NoError(t, model.CreateInstitution(ctx, db, inst))
	require.NoError(t, model.AddInstitutionToProject(ctx, db, &model.InstitutionProject{
		InstitutionID: inst.ID,
		ProjectID:     project.ID,
		Role:          "sender",
	}))

	result, err := importer.Run(ctx, db, nil, importer.Options{
		Dir:                t.TempDir(),
		DryRun:             true,
		InstitutionAETitle: "  pacs_charlie  ",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 0, result.FilesScanned)
	assert.Equal(t, 0, result.StudiesCreated)
}

func TestRun_DryRun_RejectsUnknownInstitutionAETitle(t *testing.T) {
	db := testutil.TestDB(t)

	_, err := importer.Run(context.Background(), db, nil, importer.Options{
		Dir:                t.TempDir(),
		DryRun:             true,
		InstitutionAETitle: "missing-ae",
	})
	require.Error(t, err)
	assert.True(t, importer.IsValidationError(err))
	assert.Contains(t, err.Error(), "ae_title")
	assert.Contains(t, err.Error(), "not found")
}

func TestRun_DryRun_RejectsInstitutionSlugAETitleMismatch(t *testing.T) {
	db := testutil.TestDB(t)
	ctx := context.Background()

	project, err := model.GetProjectBySlug(ctx, db, "default")
	require.NoError(t, err)

	inst := &model.Institution{
		Name:    "Hospital Delta",
		Slug:    "hospital-delta",
		Type:    "sender",
		AETitle: "PACS_DELTA",
		Enabled: true,
	}
	require.NoError(t, model.CreateInstitution(ctx, db, inst))
	require.NoError(t, model.AddInstitutionToProject(ctx, db, &model.InstitutionProject{
		InstitutionID: inst.ID,
		ProjectID:     project.ID,
		Role:          "sender",
	}))

	_, err = importer.Run(ctx, db, nil, importer.Options{
		Dir:                t.TempDir(),
		DryRun:             true,
		InstitutionSlug:    inst.Slug,
		InstitutionAETitle: "PACS_OTHER",
	})
	require.Error(t, err)
	assert.True(t, importer.IsValidationError(err))
	assert.Contains(t, err.Error(), "do not match")
}

func TestRun_DryRun_RejectsInstitutionIDAETitleMismatch(t *testing.T) {
	db := testutil.TestDB(t)
	ctx := context.Background()

	project, err := model.GetProjectBySlug(ctx, db, "default")
	require.NoError(t, err)

	inst := &model.Institution{
		Name:    "Hospital Echo",
		Slug:    "hospital-echo",
		Type:    "sender",
		AETitle: "PACS_ECHO",
		Enabled: true,
	}
	require.NoError(t, model.CreateInstitution(ctx, db, inst))
	require.NoError(t, model.AddInstitutionToProject(ctx, db, &model.InstitutionProject{
		InstitutionID: inst.ID,
		ProjectID:     project.ID,
		Role:          "sender",
	}))

	_, err = importer.Run(ctx, db, nil, importer.Options{
		Dir:                t.TempDir(),
		DryRun:             true,
		InstitutionID:      inst.ID,
		InstitutionAETitle: "PACS_OTHER",
	})
	require.Error(t, err)
	assert.True(t, importer.IsValidationError(err))
	assert.Contains(t, err.Error(), "do not match")
}

func TestRun_DryRun_RejectsAmbiguousInstitutionAETitle(t *testing.T) {
	db := testutil.TestDB(t)
	ctx := context.Background()

	project, err := model.GetProjectBySlug(ctx, db, "default")
	require.NoError(t, err)

	instA := &model.Institution{
		Name:    "Hospital Foxtrot",
		Slug:    "hospital-foxtrot",
		Type:    "sender",
		AETitle: "PACS_DUP",
		Enabled: true,
	}
	instB := &model.Institution{
		Name:    "Hospital Golf",
		Slug:    "hospital-golf",
		Type:    "sender",
		AETitle: "PACS_DUP",
		Enabled: true,
	}
	require.NoError(t, model.CreateInstitution(ctx, db, instA))
	require.NoError(t, model.CreateInstitution(ctx, db, instB))
	require.NoError(t, model.AddInstitutionToProject(ctx, db, &model.InstitutionProject{
		InstitutionID: instA.ID,
		ProjectID:     project.ID,
		Role:          "sender",
	}))
	require.NoError(t, model.AddInstitutionToProject(ctx, db, &model.InstitutionProject{
		InstitutionID: instB.ID,
		ProjectID:     project.ID,
		Role:          "sender",
	}))

	_, err = importer.Run(ctx, db, nil, importer.Options{
		Dir:                t.TempDir(),
		DryRun:             true,
		InstitutionAETitle: "pacs_dup",
	})
	require.Error(t, err)
	assert.True(t, importer.IsValidationError(err))
	assert.Contains(t, err.Error(), "matched multiple enabled institutions")
}
