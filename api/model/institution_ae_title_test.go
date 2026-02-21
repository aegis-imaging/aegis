package model_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetInstitutionByAETitle_SingleEnabledMatch(t *testing.T) {
	db := testutil.TestDB(t)
	ctx := context.Background()
	ts := time.Now().UnixNano()

	disabled := &model.Institution{
		Name:    fmt.Sprintf("Disabled %d", ts),
		Slug:    fmt.Sprintf("disabled-%d", ts),
		Type:    "sender",
		AETitle: "PACS_ALPHA",
		Enabled: false,
	}
	require.NoError(t, model.CreateInstitution(ctx, db, disabled))

	enabled := &model.Institution{
		Name:    fmt.Sprintf("Enabled %d", ts),
		Slug:    fmt.Sprintf("enabled-%d", ts),
		Type:    "sender",
		AETitle: "PACS_ALPHA",
		Enabled: true,
	}
	require.NoError(t, model.CreateInstitution(ctx, db, enabled))

	got, err := model.GetInstitutionByAETitle(ctx, db, "  pacs_alpha  ")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, enabled.ID, got.ID)
}

func TestGetInstitutionByAETitle_NotFound(t *testing.T) {
	db := testutil.TestDB(t)

	_, err := model.GetInstitutionByAETitle(context.Background(), db, "missing-ae")
	require.Error(t, err)
	assert.True(t, errors.Is(err, sql.ErrNoRows))
}

func TestGetInstitutionByAETitle_Ambiguous(t *testing.T) {
	db := testutil.TestDB(t)
	ctx := context.Background()
	ts := time.Now().UnixNano()

	instA := &model.Institution{
		Name:    fmt.Sprintf("Ambiguous A %d", ts),
		Slug:    fmt.Sprintf("ambiguous-a-%d", ts),
		Type:    "sender",
		AETitle: "PACS_DUP",
		Enabled: true,
	}
	instB := &model.Institution{
		Name:    fmt.Sprintf("Ambiguous B %d", ts),
		Slug:    fmt.Sprintf("ambiguous-b-%d", ts),
		Type:    "sender",
		AETitle: "PACS_DUP",
		Enabled: true,
	}
	require.NoError(t, model.CreateInstitution(ctx, db, instA))
	require.NoError(t, model.CreateInstitution(ctx, db, instB))

	_, err := model.GetInstitutionByAETitle(ctx, db, "pacs_dup")
	require.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrInstitutionAETitleAmbiguous))
}
