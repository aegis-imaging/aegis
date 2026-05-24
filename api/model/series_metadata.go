package model

import (
	"context"
	"database/sql"
	"time"
)

// SeriesMetadata mirrors a row in the study_series_metadata table.
//
// Numeric DICOM tags are modelled as *float64 because absence is meaningful
// (the tag may legitimately not be present on a given series) and we want to
// distinguish that from a zero value. INT-typed tags use *int for the same
// reason.
type SeriesMetadata struct {
	ID                string  `json:"id"`
	StudyID           string  `json:"study_id"`
	SeriesInstanceUID string  `json:"series_instance_uid"`
	SeriesNumber      *int    `json:"series_number,omitempty"`
	SeriesDescription *string `json:"series_description,omitempty"`
	ProtocolName      *string `json:"protocol_name,omitempty"`
	Modality          *string `json:"modality,omitempty"`
	BodyPartExamined  *string `json:"body_part_examined,omitempty"`

	// MR acquisition parameters.
	RepetitionTime         *float64 `json:"repetition_time,omitempty"`
	EchoTime               *float64 `json:"echo_time,omitempty"`
	InversionTime          *float64 `json:"inversion_time,omitempty"`
	FlipAngle              *float64 `json:"flip_angle,omitempty"`
	SliceThickness         *float64 `json:"slice_thickness,omitempty"`
	SpacingBetweenSlices   *float64 `json:"spacing_between_slices,omitempty"`
	PixelBandwidth         *float64 `json:"pixel_bandwidth,omitempty"`
	MagneticFieldStrength  *float64 `json:"magnetic_field_strength,omitempty"`
	EchoTrainLength        *int     `json:"echo_train_length,omitempty"`
	NumberOfAverages       *float64 `json:"number_of_averages,omitempty"`

	// Geometry.
	Rows            *int     `json:"rows,omitempty"`
	Columns         *int     `json:"columns,omitempty"`
	PixelSpacingRow *float64 `json:"pixel_spacing_row,omitempty"`
	PixelSpacingCol *float64 `json:"pixel_spacing_col,omitempty"`

	// Sequence identification.
	ScanningSequence  *string `json:"scanning_sequence,omitempty"`
	SequenceVariant   *string `json:"sequence_variant,omitempty"`
	MRAcquisitionType *string `json:"mr_acquisition_type,omitempty"`
	SequenceName      *string `json:"sequence_name,omitempty"`

	// Device.
	Manufacturer          *string  `json:"manufacturer,omitempty"`
	ManufacturerModelName *string  `json:"manufacturer_model_name,omitempty"`
	SoftwareVersions      *string  `json:"software_versions,omitempty"`
	ImagingFrequency      *float64 `json:"imaging_frequency,omitempty"`

	InstanceCount int       `json:"instance_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

const seriesMetadataColumns = `id, study_id, series_instance_uid, series_number,
		series_description, protocol_name, modality, body_part_examined,
		repetition_time, echo_time, inversion_time, flip_angle,
		slice_thickness, spacing_between_slices, pixel_bandwidth, magnetic_field_strength,
		echo_train_length, number_of_averages,
		rows, columns, pixel_spacing_row, pixel_spacing_col,
		scanning_sequence, sequence_variant, mr_acquisition_type, sequence_name,
		manufacturer, manufacturer_model_name, software_versions, imaging_frequency,
		instance_count, created_at, updated_at`

func scanSeriesMetadata(row interface{ Scan(...any) error }, m *SeriesMetadata) error {
	return row.Scan(
		&m.ID, &m.StudyID, &m.SeriesInstanceUID, &m.SeriesNumber,
		&m.SeriesDescription, &m.ProtocolName, &m.Modality, &m.BodyPartExamined,
		&m.RepetitionTime, &m.EchoTime, &m.InversionTime, &m.FlipAngle,
		&m.SliceThickness, &m.SpacingBetweenSlices, &m.PixelBandwidth, &m.MagneticFieldStrength,
		&m.EchoTrainLength, &m.NumberOfAverages,
		&m.Rows, &m.Columns, &m.PixelSpacingRow, &m.PixelSpacingCol,
		&m.ScanningSequence, &m.SequenceVariant, &m.MRAcquisitionType, &m.SequenceName,
		&m.Manufacturer, &m.ManufacturerModelName, &m.SoftwareVersions, &m.ImagingFrequency,
		&m.InstanceCount, &m.CreatedAt, &m.UpdatedAt,
	)
}

// UpsertSeriesMetadata inserts or updates a series metadata row, keyed by
// (study_id, series_instance_uid). On conflict every column is refreshed
// from EXCLUDED so a re-ingest of a study overwrites stale values. The ID and
// timestamps on the input struct are populated from the returning clause.
func UpsertSeriesMetadata(ctx context.Context, db *sql.DB, m *SeriesMetadata) error {
	row := db.QueryRowContext(ctx, `
		INSERT INTO study_series_metadata (
			study_id, series_instance_uid, series_number,
			series_description, protocol_name, modality, body_part_examined,
			repetition_time, echo_time, inversion_time, flip_angle,
			slice_thickness, spacing_between_slices, pixel_bandwidth, magnetic_field_strength,
			echo_train_length, number_of_averages,
			rows, columns, pixel_spacing_row, pixel_spacing_col,
			scanning_sequence, sequence_variant, mr_acquisition_type, sequence_name,
			manufacturer, manufacturer_model_name, software_versions, imaging_frequency,
			instance_count
		) VALUES (
			$1, $2, $3,
			$4, $5, $6, $7,
			$8, $9, $10, $11,
			$12, $13, $14, $15,
			$16, $17,
			$18, $19, $20, $21,
			$22, $23, $24, $25,
			$26, $27, $28, $29,
			$30
		)
		ON CONFLICT (study_id, series_instance_uid) DO UPDATE SET
			series_number           = EXCLUDED.series_number,
			series_description      = EXCLUDED.series_description,
			protocol_name           = EXCLUDED.protocol_name,
			modality                = EXCLUDED.modality,
			body_part_examined      = EXCLUDED.body_part_examined,
			repetition_time         = EXCLUDED.repetition_time,
			echo_time               = EXCLUDED.echo_time,
			inversion_time          = EXCLUDED.inversion_time,
			flip_angle              = EXCLUDED.flip_angle,
			slice_thickness         = EXCLUDED.slice_thickness,
			spacing_between_slices  = EXCLUDED.spacing_between_slices,
			pixel_bandwidth         = EXCLUDED.pixel_bandwidth,
			magnetic_field_strength = EXCLUDED.magnetic_field_strength,
			echo_train_length       = EXCLUDED.echo_train_length,
			number_of_averages      = EXCLUDED.number_of_averages,
			rows                    = EXCLUDED.rows,
			columns                 = EXCLUDED.columns,
			pixel_spacing_row       = EXCLUDED.pixel_spacing_row,
			pixel_spacing_col       = EXCLUDED.pixel_spacing_col,
			scanning_sequence       = EXCLUDED.scanning_sequence,
			sequence_variant        = EXCLUDED.sequence_variant,
			mr_acquisition_type     = EXCLUDED.mr_acquisition_type,
			sequence_name           = EXCLUDED.sequence_name,
			manufacturer            = EXCLUDED.manufacturer,
			manufacturer_model_name = EXCLUDED.manufacturer_model_name,
			software_versions       = EXCLUDED.software_versions,
			imaging_frequency       = EXCLUDED.imaging_frequency,
			instance_count          = EXCLUDED.instance_count,
			updated_at              = NOW()
		RETURNING id, created_at, updated_at`,
		m.StudyID, m.SeriesInstanceUID, m.SeriesNumber,
		m.SeriesDescription, m.ProtocolName, m.Modality, m.BodyPartExamined,
		m.RepetitionTime, m.EchoTime, m.InversionTime, m.FlipAngle,
		m.SliceThickness, m.SpacingBetweenSlices, m.PixelBandwidth, m.MagneticFieldStrength,
		m.EchoTrainLength, m.NumberOfAverages,
		m.Rows, m.Columns, m.PixelSpacingRow, m.PixelSpacingCol,
		m.ScanningSequence, m.SequenceVariant, m.MRAcquisitionType, m.SequenceName,
		m.Manufacturer, m.ManufacturerModelName, m.SoftwareVersions, m.ImagingFrequency,
		m.InstanceCount,
	)
	return row.Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
}

// ListSeriesMetadata returns every metadata row for a study, ordered by
// series_number (NULLS LAST) then by series_instance_uid for stability.
func ListSeriesMetadata(ctx context.Context, db *sql.DB, studyID string) ([]SeriesMetadata, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT `+seriesMetadataColumns+`
		FROM study_series_metadata
		WHERE study_id = $1
		ORDER BY series_number NULLS LAST, series_instance_uid`, studyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SeriesMetadata
	for rows.Next() {
		var m SeriesMetadata
		if err := scanSeriesMetadata(rows, &m); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// DeleteSeriesMetadataForStudy removes every series metadata row for a study.
// The FK from study_series_metadata.study_id to studies(id) already has
// ON DELETE CASCADE, but exposing this helper lets callers wipe just the
// metadata (e.g. before a re-ingest) without touching the parent study row.
func DeleteSeriesMetadataForStudy(ctx context.Context, db *sql.DB, studyID string) error {
	_, err := db.ExecContext(ctx,
		`DELETE FROM study_series_metadata WHERE study_id = $1`, studyID)
	return err
}
