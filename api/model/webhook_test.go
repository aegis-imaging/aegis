package model_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeWebhook(t *testing.T, projectID *string, url string, events []string) *model.WebhookSubscription {
	t.Helper()
	return &model.WebhookSubscription{
		ProjectID: projectID,
		URL:       url,
		Events:    events,
		Secret:    "s3cr3t",
		Enabled:   true,
	}
}

func TestCreateWebhookSubscription(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	w := makeWebhook(t, nil, "https://example.com/hook", []string{"study.approved", "study.rejected"})
	require.NoError(t, model.CreateWebhookSubscription(ctx, db, w))
	assert.NotEmpty(t, w.ID)
	assert.Equal(t, []string{"study.approved", "study.rejected"}, w.Events)
	assert.True(t, w.Enabled)
}

func TestGetWebhookSubscription(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	w := makeWebhook(t, nil, "https://example.com/get-hook", []string{"study.approved"})
	require.NoError(t, model.CreateWebhookSubscription(ctx, db, w))

	got, err := model.GetWebhookSubscription(ctx, db, w.ID)
	require.NoError(t, err)
	assert.Equal(t, w.ID, got.ID)
	assert.Equal(t, "https://example.com/get-hook", got.URL)
	assert.Equal(t, []string{"study.approved"}, got.Events)
}

func TestGetWebhookSubscription_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)

	_, err := model.GetWebhookSubscription(context.Background(), db, "00000000-0000-0000-0000-000000000000")
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestListWebhookSubscriptions_AllProjects(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	w1 := makeWebhook(t, nil, "https://hook.example.com/1", []string{"study.approved"})
	w2 := makeWebhook(t, nil, "https://hook.example.com/2", []string{"study.rejected"})
	require.NoError(t, model.CreateWebhookSubscription(ctx, db, w1))
	require.NoError(t, model.CreateWebhookSubscription(ctx, db, w2))

	subs, err := model.ListWebhookSubscriptions(ctx, db, "")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(subs), 2)
}

func TestListWebhookSubscriptions_ProjectScoped(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	otherProj := testutil.CreateTestProject(t, db, "wh-other-project")
	ctx := context.Background()

	projID := proj.ID
	otherID := otherProj.ID

	wProj := makeWebhook(t, &projID, "https://proj.hook.com/1", []string{"study.approved"})
	wOther := makeWebhook(t, &otherID, "https://other.hook.com/2", []string{"study.rejected"})
	require.NoError(t, model.CreateWebhookSubscription(ctx, db, wProj))
	require.NoError(t, model.CreateWebhookSubscription(ctx, db, wOther))

	subs, err := model.ListWebhookSubscriptions(ctx, db, projID)
	require.NoError(t, err)
	assert.Len(t, subs, 1)
	assert.Equal(t, wProj.ID, subs[0].ID)
}

func TestUpdateWebhookSubscription(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	w := makeWebhook(t, nil, "https://old.url.com", []string{"study.approved"})
	require.NoError(t, model.CreateWebhookSubscription(ctx, db, w))

	w.URL = "https://new.url.com"
	w.Events = []string{"study.approved", "study.phi_flagged"}
	w.Enabled = false
	require.NoError(t, model.UpdateWebhookSubscription(ctx, db, w))

	got, err := model.GetWebhookSubscription(ctx, db, w.ID)
	require.NoError(t, err)
	assert.Equal(t, "https://new.url.com", got.URL)
	assert.Equal(t, []string{"study.approved", "study.phi_flagged"}, got.Events)
	assert.False(t, got.Enabled)
}

func TestDeleteWebhookSubscription(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	w := makeWebhook(t, nil, "https://delete.me", []string{"study.approved"})
	require.NoError(t, model.CreateWebhookSubscription(ctx, db, w))

	require.NoError(t, model.DeleteWebhookSubscription(ctx, db, w.ID))

	_, err := model.GetWebhookSubscription(ctx, db, w.ID)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestRecordWebhookDelivery_AndList(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	w := makeWebhook(t, nil, "https://delivery.example.com", []string{"study.approved"})
	require.NoError(t, model.CreateWebhookSubscription(ctx, db, w))

	code := 200
	model.RecordWebhookDelivery(ctx, db, &model.WebhookDelivery{
		SubscriptionID: w.ID,
		Event:          "study.approved",
		URL:            w.URL,
		Attempt:        1,
		StatusCode:     &code,
		Success:        true,
	})

	code2 := 500
	model.RecordWebhookDelivery(ctx, db, &model.WebhookDelivery{
		SubscriptionID: w.ID,
		Event:          "study.approved",
		URL:            w.URL,
		Attempt:        2,
		StatusCode:     &code2,
		Success:        false,
	})

	deliveries, err := model.ListWebhookDeliveries(ctx, db, w.ID, 50)
	require.NoError(t, err)
	assert.Len(t, deliveries, 2)
	// Newest first — attempt 2 should be first.
	assert.Equal(t, 2, deliveries[0].Attempt)
	assert.Equal(t, 1, deliveries[1].Attempt)
}

func TestGetWebhookStats(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	w := makeWebhook(t, nil, "https://stats.example.com", []string{"study.approved"})
	require.NoError(t, model.CreateWebhookSubscription(ctx, db, w))

	code200 := 200
	code500 := 500
	model.RecordWebhookDelivery(ctx, db, &model.WebhookDelivery{
		SubscriptionID: w.ID, Event: "study.approved", URL: w.URL, Attempt: 1,
		StatusCode: &code200, Success: true,
	})
	model.RecordWebhookDelivery(ctx, db, &model.WebhookDelivery{
		SubscriptionID: w.ID, Event: "study.rejected", URL: w.URL, Attempt: 1,
		StatusCode: &code500, Success: false,
	})

	stats, err := model.GetWebhookStats(ctx, db, w.ID)
	require.NoError(t, err)
	assert.Equal(t, w.ID, stats.SubscriptionID)
	assert.Equal(t, 2, stats.TotalDeliveries)
	assert.Equal(t, 1, stats.Successful)
	assert.Equal(t, 1, stats.Failed)
	assert.InDelta(t, 50.0, stats.SuccessRatePct, 0.01)
	assert.NotNil(t, stats.LastDeliveryAt)
	assert.Equal(t, 1, stats.DeliveriesByEvent["study.approved"])
	assert.Equal(t, 1, stats.DeliveriesByEvent["study.rejected"])
}

func TestListEnabledWebhooksForEvent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	// Global subscription (project_id IS NULL) — matches any project.
	wMatch := makeWebhook(t, nil, "https://match.example.com", []string{"study.approved", "study.rejected"})
	// Global subscription that doesn't subscribe to the queried event.
	wNoMatch := makeWebhook(t, nil, "https://nomatch.example.com", []string{"study.export_complete"})
	// Global subscription for the right event but disabled.
	wDisabled := makeWebhook(t, nil, "https://disabled.example.com", []string{"study.approved"})
	wDisabled.Enabled = false

	require.NoError(t, model.CreateWebhookSubscription(ctx, db, wMatch))
	require.NoError(t, model.CreateWebhookSubscription(ctx, db, wNoMatch))
	require.NoError(t, model.CreateWebhookSubscription(ctx, db, wDisabled))

	// Function is always called with a valid project UUID (study's ProjectID).
	subs, err := model.ListEnabledWebhooksForEvent(ctx, db, "study.approved", proj.ID)
	require.NoError(t, err)

	var ids []string
	for _, s := range subs {
		ids = append(ids, s.ID)
	}
	assert.Contains(t, ids, wMatch.ID)
	assert.NotContains(t, ids, wNoMatch.ID)
	assert.NotContains(t, ids, wDisabled.ID)
}

func TestListAllWebhookDeliveries_SuccessFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	w := makeWebhook(t, nil, "https://filter.example.com", []string{"study.approved"})
	require.NoError(t, model.CreateWebhookSubscription(ctx, db, w))

	code200 := 200
	code400 := 400
	model.RecordWebhookDelivery(ctx, db, &model.WebhookDelivery{
		SubscriptionID: w.ID, Event: "study.approved", URL: w.URL,
		Attempt: 1, StatusCode: &code200, Success: true,
	})
	model.RecordWebhookDelivery(ctx, db, &model.WebhookDelivery{
		SubscriptionID: w.ID, Event: "study.approved", URL: w.URL,
		Attempt: 2, StatusCode: &code400, Success: false,
	})

	successOnly := true
	deliveries, total, err := model.ListAllWebhookDeliveries(ctx, db, w.ID, &successOnly, 50, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, deliveries, 1)
	assert.True(t, deliveries[0].Success)
}
