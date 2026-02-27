package retention
package retention

import (
	"context"
	"testing"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunExpiresOnlyApprovedStudiesOlderThanPolicy(t *testing.T) {
	db := testutil.TestDB(t)
	ctx := context.Background()
	project := testutil.SeedProject(t, db)

	retentionDays := 30
	err := model.UpdateProjectRetentionDays(ctx, db, project.ID, &retentionDays)
	require.NoError(t, err)

	oldApproved := testutil.CreateTestStudy(t, db, project.ID)
	newApproved := testutil.CreateTestStudy(t, db, project.ID)
	oldReceived := testutil.CreateTestStudy(t, db, project.ID)

	_, err = db.ExecContext(ctx, `UPDATE studies SET status = 'approved', created_at = $1 WHERE id = $2`, time.Now().UTC().AddDate(0, 0, -40), oldApproved.ID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `UPDATE studies SET status = 'approved', created_at = $1 WHERE id = $2`, time.Now().UTC().AddDate(0, 0, -5), newApproved.ID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `UPDATE studies SET status = 'received', created_at = $1 WHERE id = $2`, time.Now().UTC().AddDate(0, 0, -40), oldReceived.ID)
	require.NoError(t, err)

	run(ctx, db)

	updatedOldApproved, err := model.GetStudyByID(ctx, db, oldApproved.ID)
	require.NoError(t, err)
	updatedNewApproved, err := model.GetStudyByID(ctx, db, newApproved.ID)
	require.NoError(t, err)
	updatedOldReceived, err := model.GetStudyByID(ctx, db, oldReceived.ID)
	require.NoError(t, err)

	assert.Equal(t, "expired", updatedOldApproved.Status)
	assert.Equal(t, "approved", updatedNewApproved.Status)
	assert.Equal(t, "received", updatedOldReceived.Status)
}