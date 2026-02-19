package model_test

import (
	"context"
	"testing"

	"github.com/msenjem/aegis/api/model"
	"github.com/msenjem/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateDestination(t *testing.T) {
	db := testutil.TestDB(t)

	d := testutil.CreateTestDestination(t, db, "test-dest")
	assert.NotEmpty(t, d.ID)
	assert.Equal(t, "test-dest", d.Name)
	assert.Equal(t, "dicomweb", d.Type)
	assert.True(t, d.Enabled)
}

func TestListDestinations(t *testing.T) {
	db := testutil.TestDB(t)
	testutil.CreateTestDestination(t, db, "alpha")
	testutil.CreateTestDestination(t, db, "beta")

	dests, err := model.ListDestinations(context.Background(), db)
	require.NoError(t, err)
	assert.Len(t, dests, 2)
	assert.Equal(t, "alpha", dests[0].Name) // ordered by name
}

func TestCreateRoutingRule(t *testing.T) {
	db := testutil.TestDB(t)

	r := testutil.CreateTestRoutingRule(t, db, "test-rule", "require_defacing")
	assert.NotEmpty(t, r.ID)
	assert.Equal(t, "require_defacing", r.Action)
	assert.True(t, r.Enabled)
}

func TestListEnabledRoutingRules(t *testing.T) {
	db := testutil.TestDB(t)

	r1 := &model.RoutingRule{Name: "low", Priority: 10, Enabled: true, Action: "require_qa"}
	r2 := &model.RoutingRule{Name: "high", Priority: 1, Enabled: true, Action: "require_defacing"}
	r3 := &model.RoutingRule{Name: "disabled", Priority: 5, Enabled: false, Action: "reject"}

	require.NoError(t, model.CreateRoutingRule(context.Background(), db, r1))
	require.NoError(t, model.CreateRoutingRule(context.Background(), db, r2))
	require.NoError(t, model.CreateRoutingRule(context.Background(), db, r3))

	rules, err := model.ListEnabledRoutingRules(context.Background(), db)
	require.NoError(t, err)
	assert.Len(t, rules, 2, "disabled rule should be excluded")
	assert.Equal(t, "high", rules[0].Name, "lower priority number should come first")
	assert.Equal(t, "low", rules[1].Name)
}

func TestCreateRoutingLogEntry(t *testing.T) {
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)
	rule := testutil.CreateTestRoutingRule(t, db, "log-test", "require_qa")

	entry := &model.RoutingLogEntry{
		StudyID: study.ID,
		RuleID:  rule.ID,
		Action:  "require_qa",
		Outcome: "acknowledged",
	}
	err := model.CreateRoutingLogEntry(context.Background(), db, entry)
	require.NoError(t, err)
	assert.NotEmpty(t, entry.ID)

	log, err := model.ListRoutingLogForStudy(context.Background(), db, study.ID)
	require.NoError(t, err)
	assert.Len(t, log, 1)
	assert.Equal(t, "require_qa", log[0].Action)
}
