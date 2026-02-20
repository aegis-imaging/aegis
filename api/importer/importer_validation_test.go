package importer

import (
	"context"
	"testing"

	"github.com/msenjem/aegis/api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateImportInstitution_ValidSender(t *testing.T) {
	err := validateImportInstitution(&model.Institution{
		ID:      "inst-1",
		Type:    "sender",
		Enabled: true,
	})
	require.NoError(t, err)
}

func TestValidateImportInstitution_ValidBoth(t *testing.T) {
	err := validateImportInstitution(&model.Institution{
		ID:      "inst-2",
		Type:    "both",
		Enabled: true,
	})
	require.NoError(t, err)
}

func TestValidateImportInstitution_RejectsNil(t *testing.T) {
	err := validateImportInstitution(nil)
	require.Error(t, err)
	assert.True(t, IsValidationError(err))
}

func TestValidateImportInstitution_RejectsDisabled(t *testing.T) {
	err := validateImportInstitution(&model.Institution{
		ID:      "inst-3",
		Type:    "sender",
		Enabled: false,
	})
	require.Error(t, err)
	assert.True(t, IsValidationError(err))
}

func TestValidateImportInstitution_RejectsReceiverOnly(t *testing.T) {
	err := validateImportInstitution(&model.Institution{
		ID:      "inst-4",
		Type:    "receiver",
		Enabled: true,
	})
	require.Error(t, err)
	assert.True(t, IsValidationError(err))
}

func TestValidationErrorMarker(t *testing.T) {
	err := validationErrorf("invalid project")
	assert.True(t, IsValidationError(err))
}

func TestNormalizeInstitutionSelectors_TrimAndLower(t *testing.T) {
	opts := &Options{
		InstitutionID:   "  ",
		InstitutionSlug: "  Hospital-BRAVO  ",
	}

	err := normalizeInstitutionSelectors(opts)
	require.NoError(t, err)
	assert.Equal(t, "", opts.InstitutionID)
	assert.Equal(t, "hospital-bravo", opts.InstitutionSlug)
}

func TestNormalizeInstitutionSelectors_RejectsConflictingSelectors(t *testing.T) {
	opts := &Options{
		InstitutionID:   "inst-1",
		InstitutionSlug: "hospital-bravo",
	}

	err := normalizeInstitutionSelectors(opts)
	require.Error(t, err)
	assert.True(t, IsValidationError(err))
	assert.Contains(t, err.Error(), "only one")
}

func TestRun_RejectsConflictingInstitutionSelectors(t *testing.T) {
	_, err := Run(context.Background(), nil, nil, Options{
		InstitutionID:   "inst-1",
		InstitutionSlug: "hospital-bravo",
	})
	require.Error(t, err)
	assert.True(t, IsValidationError(err))
	assert.Contains(t, err.Error(), "only one")
}
