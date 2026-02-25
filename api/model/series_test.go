package model_test

import (
	"context"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpsertStudySeries_Insert(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	s := &model.StudySeries{
		StudyID:           study.ID,
		SeriesInstanceUID: "1.2.3.4.5.6.100",
		SeriesDescription: "T1w MPRAGE",
		Modality:          "MR",
		BodyPart:          "HEAD",
		InstanceCount:     176,
	}
	require.NoError(t, model.UpsertStudySeries(context.Background(), db, s))

	series, err := model.ListStudySeries(context.Background(), db, study.ID)
	require.NoError(t, err)
	require.Len(t, series, 1)
	assert.Equal(t, "1.2.3.4.5.6.100", series[0].SeriesInstanceUID)
	assert.Equal(t, "T1w MPRAGE", series[0].SeriesDescription)
	assert.Equal(t, "MR", series[0].Modality)
	assert.Equal(t, "HEAD", series[0].BodyPart)
	assert.Equal(t, 176, series[0].InstanceCount)
	assert.Equal(t, study.ID, series[0].StudyID)
}

func TestUpsertStudySeries_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)
	ctx := context.Background()

	s := &model.StudySeries{
		StudyID:           study.ID,
		SeriesInstanceUID: "1.2.3.4.5.6.200",
		Modality:          "MR",
		InstanceCount:     10,
	}
	require.NoError(t, model.UpsertStudySeries(ctx, db, s))

	// Upsert again with updated fields.
	s2 := &model.StudySeries{
		StudyID:           study.ID,
		SeriesInstanceUID: "1.2.3.4.5.6.200",
		SeriesDescription: "Updated description",
		Modality:          "MR",
		BodyPart:          "CHEST",
		InstanceCount:     55,
	}
	require.NoError(t, model.UpsertStudySeries(ctx, db, s2))

	series, err := model.ListStudySeries(ctx, db, study.ID)
	require.NoError(t, err)
	require.Len(t, series, 1, "upsert should not insert a duplicate")
	assert.Equal(t, "Updated description", series[0].SeriesDescription)
	assert.Equal(t, "CHEST", series[0].BodyPart)
	assert.Equal(t, 55, series[0].InstanceCount)
}

func TestListStudySeries_MultipleSeries_OrderedByUID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)
	ctx := context.Background()

	for _, uid := range []string{"1.2.3.4.300", "1.2.3.4.100", "1.2.3.4.200"} {
		require.NoError(t, model.UpsertStudySeries(ctx, db, &model.StudySeries{
			StudyID:           study.ID,
			SeriesInstanceUID: uid,
			Modality:          "MR",
			InstanceCount:     1,
		}))
	}

	series, err := model.ListStudySeries(ctx, db, study.ID)
	require.NoError(t, err)
	assert.Len(t, series, 3)
	// Ordered by series_instance_uid ASC.
	assert.Equal(t, "1.2.3.4.100", series[0].SeriesInstanceUID)
	assert.Equal(t, "1.2.3.4.200", series[1].SeriesInstanceUID)
	assert.Equal(t, "1.2.3.4.300", series[2].SeriesInstanceUID)
}

func TestListStudySeries_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	series, err := model.ListStudySeries(context.Background(), db, study.ID)
	require.NoError(t, err)
	assert.Empty(t, series)
}
