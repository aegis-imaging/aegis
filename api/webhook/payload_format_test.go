package webhook

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPostTestDeliversAegisEnvelope verifies the legacy AEGIS-envelope path
// posts the full Payload struct as the request body. This is the default
// for any subscription created without payload_format set.
func TestPostTestDeliversAegisEnvelope(t *testing.T) {
	var receivedBody []byte
	var receivedContentType string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		receivedBody = b
		receivedContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	fhir := buildFHIRImagingStudy(&model.Study{
		StudyInstanceUID: "1.2.3.4",
		Status:           "approved",
		Modality:         "MRI",
	})
	payload := Payload{
		Event:            "study.approved",
		StudyID:          "abc",
		StudyInstanceUID: "1.2.3.4",
		ProjectID:        "p",
		Timestamp:        "2026-05-23T00:00:00Z",
		FHIRImagingStudy: &fhir,
	}
	sub := model.WebhookSubscription{URL: srv.URL, PayloadFormat: ""} // empty = aegis

	status, err := PostTest(sub, payload)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "application/json", receivedContentType)

	var got Payload
	require.NoError(t, json.Unmarshal(receivedBody, &got))
	assert.Equal(t, "study.approved", got.Event)
	assert.NotNil(t, got.FHIRImagingStudy)
	assert.Equal(t, "ImagingStudy", got.FHIRImagingStudy.ResourceType)
}

// TestPostTestDeliversBareFHIRBody verifies a subscription with
// payload_format=fhir receives a top-level FHIR ImagingStudy resource as
// the body — no AEGIS envelope, no `event` field, no `study_id` field.
// This is the contract EMR/RIS systems that speak only FHIR depend on.
func TestPostTestDeliversBareFHIRBody(t *testing.T) {
	var receivedBody []byte
	var receivedContentType string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		receivedBody = b
		receivedContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	fhir := buildFHIRImagingStudy(&model.Study{
		StudyInstanceUID: "1.2.3.4",
		Status:           "approved",
		Modality:         "MRI",
		SeriesCount:      3,
		InstanceCount:    150,
		StudyDescription: "Brain MRI",
	})
	payload := Payload{
		Event:            "study.approved",
		StudyID:          "abc",
		StudyInstanceUID: "1.2.3.4",
		FHIRImagingStudy: &fhir,
	}
	sub := model.WebhookSubscription{URL: srv.URL, PayloadFormat: model.PayloadFormatFHIR}

	status, err := PostTest(sub, payload)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "application/fhir+json", receivedContentType)

	// The body should be a bare FHIRImagingStudy — top-level resourceType.
	var got FHIRImagingStudy
	require.NoError(t, json.Unmarshal(receivedBody, &got))
	assert.Equal(t, "ImagingStudy", got.ResourceType)
	assert.Equal(t, "1.2.3.4", got.ID)
	assert.Equal(t, 3, got.NumberOfSeries)
	assert.Equal(t, 150, got.NumberOfInstances)

	// And it must NOT be the AEGIS envelope.
	var asEnvelope map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(receivedBody, &asEnvelope))
	_, hasEvent := asEnvelope["event"]
	_, hasStudyID := asEnvelope["study_id"]
	assert.False(t, hasEvent, "FHIR body must not include AEGIS envelope 'event' field")
	assert.False(t, hasStudyID, "FHIR body must not include AEGIS envelope 'study_id' field")
}

func TestPostTestFHIRRequiresFHIRImagingStudy(t *testing.T) {
	sub := model.WebhookSubscription{URL: "http://unused", PayloadFormat: model.PayloadFormatFHIR}
	_, err := PostTest(sub, Payload{Event: "study.approved"}) // no FHIRImagingStudy
	assert.ErrorContains(t, err, "fhir_imaging_study")
}

func TestContentTypeForFormat(t *testing.T) {
	assert.Equal(t, "application/json", contentTypeFor(""))
	assert.Equal(t, "application/json", contentTypeFor(model.PayloadFormatAegis))
	assert.Equal(t, "application/json", contentTypeFor("unknown-value"))
	assert.Equal(t, "application/fhir+json", contentTypeFor(model.PayloadFormatFHIR))
}

func TestNormalizePayloadFormat(t *testing.T) {
	assert.Equal(t, model.PayloadFormatAegis, model.NormalizePayloadFormat(""))
	assert.Equal(t, model.PayloadFormatAegis, model.NormalizePayloadFormat("aegis"))
	assert.Equal(t, model.PayloadFormatAegis, model.NormalizePayloadFormat("unknown"))
	assert.Equal(t, model.PayloadFormatFHIR, model.NormalizePayloadFormat("fhir"))
}
