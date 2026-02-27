package webhook

import (
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/stretchr/testify/assert"
)

func TestFHIRStudyStatusMapping(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "approved", want: "available"},
		{in: "received", want: "registered"},
		{in: "defacing", want: "registered"},
		{in: "clean", want: "registered"},
		{in: "defaced", want: "registered"},
		{in: "rejected", want: "cancelled"},
		{in: "expired", want: "cancelled"},
		{in: "other", want: "unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			assert.Equal(t, tc.want, fhirStudyStatus(tc.in))
		})
	}
}

func TestBuildFHIRImagingStudyWithModality(t *testing.T) {
	study := &model.Study{
		StudyInstanceUID: "1.2.840.113619.2.55.3.604688435.123.456",
		Status:           "approved",
		Modality:         "MRI",
		SeriesCount:      2,
		InstanceCount:    42,
		StudyDescription: "Brain MRI",
	}

	out := buildFHIRImagingStudy(study)

	assert.Equal(t, "ImagingStudy", out.ResourceType)
	assert.Equal(t, study.StudyInstanceUID, out.ID)
	assert.Equal(t, "available", out.Status)
	assert.Equal(t, "Patient/unknown", out.Subject.Reference)
	assert.Len(t, out.Identifier, 1)
	assert.Equal(t, "urn:dicom:uid", out.Identifier[0].System)
	assert.Equal(t, "urn:oid:"+study.StudyInstanceUID, out.Identifier[0].Value)
	assert.Equal(t, study.SeriesCount, out.NumberOfSeries)
	assert.Equal(t, study.InstanceCount, out.NumberOfInstances)
	assert.Equal(t, study.StudyDescription, out.Description)
	assert.Len(t, out.Modality, 1)
	assert.Len(t, out.Modality[0].Coding, 1)
	assert.Equal(t, "http://dicom.nema.org/resources/ontology/DCM", out.Modality[0].Coding[0].System)
	assert.Equal(t, "MRI", out.Modality[0].Coding[0].Code)
}

func TestBuildFHIRImagingStudyWithoutModality(t *testing.T) {
	study := &model.Study{
		StudyInstanceUID: "1.2.3.4.5",
		Status:           "received",
		Modality:         "",
	}

	out := buildFHIRImagingStudy(study)

	assert.Empty(t, out.Modality)
	assert.Equal(t, "registered", out.Status)
}
