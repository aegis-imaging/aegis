package audit_retention
package audit_retention

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

func TestPurgeDeletesOnlyOldAuditRows(t *testing.T) {
	db := testutil.TestDB(t)
	ctx := context.Background()

	oldResource := fmt.Sprintf("old-%d", time.Now().UnixNano())
	newResource := fmt.Sprintf("new-%d", time.Now().UnixNano())

	err := model.CreateAuditEntry(ctx, db, "study.created", "tester", "study", oldResource, "127.0.0.1", nil)
	require.NoError(t, err)
	err = model.CreateAuditEntry(ctx, db, "study.created", "tester", "study", newResource, "127.0.0.1", nil)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `UPDATE audit_trail SET created_at = $1 WHERE resource_id = $2`, time.Now().UTC().AddDate(0, 0, -10), oldResource)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `UPDATE audit_trail SET created_at = $1 WHERE resource_id = $2`, time.Now().UTC().AddDate(0, 0, -1), newResource)
	require.NoError(t, err)

	purge(ctx, db, 7)

	var oldCount int
	err = db.QueryRowContext(ctx, `SELECT count(*) FROM audit_trail WHERE resource_id = $1`, oldResource).Scan(&oldCount)
	require.NoError(t, err)
	assert.Equal(t, 0, oldCount)

	var newCount int
	err = db.QueryRowContext(ctx, `SELECT count(*) FROM audit_trail WHERE resource_id = $1`, newResource).Scan(&newCount)
	require.NoError(t, err)
	assert.Equal(t, 1, newCount)
}