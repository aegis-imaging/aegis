package webhook

import "github.com/aegis-imaging/aegis/api/model"

// FHIR R4 ImagingStudy resource types — minimal subset needed for the webhook payload.
// Spec: https://www.hl7.org/fhir/R4/imagingstudy.html

type fhirCoding struct {
	System string `json:"system"`
	Code   string `json:"code"`
}

type fhirCodeableConcept struct {
	Coding []fhirCoding `json:"coding"`
}

type fhirReference struct {
	Reference string `json:"reference"`
}

type fhirIdentifier struct {
	System string `json:"system"`
	Value  string `json:"value"`
}

// FHIRImagingStudy is a minimal FHIR R4 ImagingStudy resource. No PHI is
// included — only DICOM metadata that has already passed through the
// anonymization pipeline (UID, modality, series/instance counts, description).
type FHIRImagingStudy struct {
	ResourceType      string                `json:"resourceType"`
	ID                string                `json:"id"`
	Status            string                `json:"status"`
	Subject           fhirReference         `json:"subject"`
	Identifier        []fhirIdentifier      `json:"identifier"`
	Modality          []fhirCodeableConcept `json:"modality,omitempty"`
	NumberOfSeries    int                   `json:"numberOfSeries,omitempty"`
	NumberOfInstances int                   `json:"numberOfInstances,omitempty"`
	Description       string                `json:"description,omitempty"`
}

// buildFHIRImagingStudy converts an AEGIS study record into a minimal FHIR R4
// ImagingStudy resource for inclusion in webhook payloads.
func buildFHIRImagingStudy(study *model.Study) FHIRImagingStudy {
	fhir := FHIRImagingStudy{
		ResourceType: "ImagingStudy",
		ID:           study.StudyInstanceUID,
		Status:       fhirStudyStatus(study.Status),
		Subject:      fhirReference{Reference: "Patient/unknown"},
		Identifier: []fhirIdentifier{
			{
				System: "urn:dicom:uid",
				Value:  "urn:oid:" + study.StudyInstanceUID,
			},
		},
		NumberOfSeries:    study.SeriesCount,
		NumberOfInstances: study.InstanceCount,
		Description:       study.StudyDescription,
	}

	if study.Modality != "" {
		fhir.Modality = []fhirCodeableConcept{
			{
				Coding: []fhirCoding{
					{
						System: "http://dicom.nema.org/resources/ontology/DCM",
						Code:   study.Modality,
					},
				},
			},
		}
	}

	return fhir
}

// fhirStudyStatus maps an AEGIS study status to the FHIR ImagingStudy.status
// value set (available | registered | cancelled | unknown).
func fhirStudyStatus(status string) string {
	switch status {
	case "approved":
		return "available"
	case "received", "defacing", "clean", "defaced":
		return "registered"
	case "rejected", "expired":
		return "cancelled"
	default:
		return "unknown"
	}
}
