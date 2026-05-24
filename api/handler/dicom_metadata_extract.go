package handler

import (
	"strconv"
	"strings"

	dicomlib "github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/tag"

	"github.com/aegis-imaging/aegis/api/model"
)

// extractSeriesMetadata pulls the tags listed in protocol-service's extractor
// from a single DICOM dataset and returns a populated SeriesMetadata. The
// caller is responsible for setting StudyID and InstanceCount before upsert.
//
// Every nullable field is populated only when the tag is present and parses
// successfully so that the resulting row reflects what the scanner reported
// (an absent tag stays NULL in the DB, not 0).
func extractSeriesMetadata(dataset dicomlib.Dataset) *model.SeriesMetadata {
	m := &model.SeriesMetadata{
		SeriesInstanceUID: getStringTag(dataset, tag.SeriesInstanceUID),
	}

	// Identification.
	m.SeriesNumber = getIntTag(dataset, tag.SeriesNumber)
	m.SeriesDescription = nullableStringTag(dataset, tag.SeriesDescription)
	m.ProtocolName = nullableStringTag(dataset, tag.ProtocolName)
	m.Modality = nullableStringTag(dataset, tag.Modality)
	m.BodyPartExamined = nullableStringTag(dataset, tag.BodyPartExamined)

	// MR acquisition parameters (DS = decimal string, IS = integer string).
	m.RepetitionTime = getFloatTag(dataset, tag.RepetitionTime)
	m.EchoTime = getFloatTag(dataset, tag.EchoTime)
	m.InversionTime = getFloatTag(dataset, tag.InversionTime)
	m.FlipAngle = getFloatTag(dataset, tag.FlipAngle)
	m.SliceThickness = getFloatTag(dataset, tag.SliceThickness)
	m.SpacingBetweenSlices = getFloatTag(dataset, tag.SpacingBetweenSlices)
	m.PixelBandwidth = getFloatTag(dataset, tag.PixelBandwidth)
	m.MagneticFieldStrength = getFloatTag(dataset, tag.MagneticFieldStrength)
	m.EchoTrainLength = getIntTag(dataset, tag.EchoTrainLength)
	m.NumberOfAverages = getFloatTag(dataset, tag.NumberOfAverages)

	// Geometry. Rows/Columns are US (Unsigned Short) so they come back as ints.
	m.Rows = getIntTag(dataset, tag.Rows)
	m.Columns = getIntTag(dataset, tag.Columns)

	// PixelSpacing is DS multi-valued: row spacing first, then column.
	if rowSpacing, colSpacing, ok := getFloatPairTag(dataset, tag.PixelSpacing); ok {
		m.PixelSpacingRow = &rowSpacing
		m.PixelSpacingCol = &colSpacing
	}

	// Sequence identification.
	m.ScanningSequence = nullableStringTag(dataset, tag.ScanningSequence)
	m.SequenceVariant = nullableStringTag(dataset, tag.SequenceVariant)
	m.MRAcquisitionType = nullableStringTag(dataset, tag.MRAcquisitionType)
	m.SequenceName = nullableStringTag(dataset, tag.SequenceName)

	// Device.
	m.Manufacturer = nullableStringTag(dataset, tag.Manufacturer)
	m.ManufacturerModelName = nullableStringTag(dataset, tag.ManufacturerModelName)
	m.SoftwareVersions = joinedStringsTag(dataset, tag.SoftwareVersions)
	m.ImagingFrequency = getFloatTag(dataset, tag.ImagingFrequency)

	return m
}

// getStringTag returns the first string value of a tag, or "" when absent. The
// existing handler package already has stowGetStringTag with identical
// semantics, but the extractor was written to be importable from places that
// don't depend on the STOW receiver, so the helper is duplicated here under a
// non-stow name.
func getStringTag(dataset dicomlib.Dataset, t tag.Tag) string {
	el, err := dataset.FindElementByTag(t)
	if err != nil || el.Value == nil {
		return ""
	}
	strs, ok := el.Value.GetValue().([]string)
	if !ok || len(strs) == 0 {
		return ""
	}
	return strings.TrimSpace(strs[0])
}

// nullableStringTag returns a *string that is nil when the tag is missing or
// empty after trimming. This preserves the "absent vs empty" distinction in
// the database (NULL vs "").
func nullableStringTag(dataset dicomlib.Dataset, t tag.Tag) *string {
	s := getStringTag(dataset, t)
	if s == "" {
		return nil
	}
	return &s
}

// joinedStringsTag concatenates a multi-valued string tag with '\' separators
// (matching the DICOM serialization). Returns nil when absent or empty after
// trimming each component.
func joinedStringsTag(dataset dicomlib.Dataset, t tag.Tag) *string {
	el, err := dataset.FindElementByTag(t)
	if err != nil || el.Value == nil {
		return nil
	}
	strs, ok := el.Value.GetValue().([]string)
	if !ok || len(strs) == 0 {
		return nil
	}
	clean := make([]string, 0, len(strs))
	for _, s := range strs {
		s = strings.TrimSpace(s)
		if s != "" {
			clean = append(clean, s)
		}
	}
	if len(clean) == 0 {
		return nil
	}
	joined := strings.Join(clean, `\`)
	return &joined
}

// getFloatTag parses a DICOM tag that may be DS, FL, FD, IS, or US into a
// float64. Returns nil when the tag is missing or unparseable.
func getFloatTag(dataset dicomlib.Dataset, t tag.Tag) *float64 {
	el, err := dataset.FindElementByTag(t)
	if err != nil || el.Value == nil {
		return nil
	}
	switch v := el.Value.GetValue().(type) {
	case []string:
		if len(v) == 0 {
			return nil
		}
		s := strings.TrimSpace(v[0])
		if s == "" {
			return nil
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil
		}
		return &f
	case []float64:
		if len(v) == 0 {
			return nil
		}
		f := v[0]
		return &f
	case []float32:
		if len(v) == 0 {
			return nil
		}
		f := float64(v[0])
		return &f
	case []int:
		if len(v) == 0 {
			return nil
		}
		f := float64(v[0])
		return &f
	}
	return nil
}

// getFloatPairTag parses a multi-valued DS tag (e.g. PixelSpacing) into the
// first two numeric components. Returns ok=false if fewer than two values are
// present or any value fails to parse.
func getFloatPairTag(dataset dicomlib.Dataset, t tag.Tag) (float64, float64, bool) {
	el, err := dataset.FindElementByTag(t)
	if err != nil || el.Value == nil {
		return 0, 0, false
	}
	switch v := el.Value.GetValue().(type) {
	case []string:
		if len(v) < 2 {
			return 0, 0, false
		}
		a, err := strconv.ParseFloat(strings.TrimSpace(v[0]), 64)
		if err != nil {
			return 0, 0, false
		}
		b, err := strconv.ParseFloat(strings.TrimSpace(v[1]), 64)
		if err != nil {
			return 0, 0, false
		}
		return a, b, true
	case []float64:
		if len(v) < 2 {
			return 0, 0, false
		}
		return v[0], v[1], true
	case []float32:
		if len(v) < 2 {
			return 0, 0, false
		}
		return float64(v[0]), float64(v[1]), true
	}
	return 0, 0, false
}

// getIntTag parses a DICOM tag that may be IS (integer string) or US/SS/UL/SL
// (binary int) into a Go int. Returns nil when absent or unparseable.
func getIntTag(dataset dicomlib.Dataset, t tag.Tag) *int {
	el, err := dataset.FindElementByTag(t)
	if err != nil || el.Value == nil {
		return nil
	}
	switch v := el.Value.GetValue().(type) {
	case []string:
		if len(v) == 0 {
			return nil
		}
		s := strings.TrimSpace(v[0])
		if s == "" {
			return nil
		}
		i, err := strconv.Atoi(s)
		if err != nil {
			// Some scanners write "1.0" in an IS slot; tolerate that by
			// rounding the float.
			if f, ferr := strconv.ParseFloat(s, 64); ferr == nil {
				ii := int(f)
				return &ii
			}
			return nil
		}
		return &i
	case []int:
		if len(v) == 0 {
			return nil
		}
		i := v[0]
		return &i
	case []uint32:
		if len(v) == 0 {
			return nil
		}
		i := int(v[0])
		return &i
	case []uint16:
		if len(v) == 0 {
			return nil
		}
		i := int(v[0])
		return &i
	case []int16:
		if len(v) == 0 {
			return nil
		}
		i := int(v[0])
		return &i
	case []int32:
		if len(v) == 0 {
			return nil
		}
		i := int(v[0])
		return &i
	}
	return nil
}
