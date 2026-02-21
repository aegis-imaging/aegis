package importer

import (
	"context"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
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
		Dir:             "/tmp",
		InstitutionID:   "inst-1",
		InstitutionSlug: "hospital-bravo",
	})
	require.Error(t, err)
	assert.True(t, IsValidationError(err))
	assert.Contains(t, err.Error(), "only one")
}

func TestNormalizeImportDir_Required(t *testing.T) {
	opts := &Options{Dir: "   "}
	err := normalizeImportDir(opts)
	require.Error(t, err)
	assert.True(t, IsValidationError(err))
	assert.Contains(t, err.Error(), "dir is required")
}

func TestNormalizeImportDir_TrimAndClean(t *testing.T) {
	opts := &Options{Dir: " /tmp/../tmp "}
	require.NoError(t, normalizeImportDir(opts))
	assert.Equal(t, "/tmp", opts.Dir)
}

func TestNormalizeImportDir_RejectsRelativePath(t *testing.T) {
	opts := &Options{Dir: " ./testdata "}
	err := normalizeImportDir(opts)
	require.Error(t, err)
	assert.True(t, IsValidationError(err))
	assert.Contains(t, err.Error(), "absolute path")
}

func TestNormalizeProjectSlug_DefaultsToDefault(t *testing.T) {
	opts := &Options{ProjectSlug: "   "}
	normalizeProjectSlug(opts)
	assert.Equal(t, "default", opts.ProjectSlug)
}

func TestNormalizeProjectSlug_TrimAndLower(t *testing.T) {
	opts := &Options{ProjectSlug: "  ReSearch-Study  "}
	normalizeProjectSlug(opts)
	assert.Equal(t, "research-study", opts.ProjectSlug)
}

func TestNormalizeImportSource_DefaultsToInternal(t *testing.T) {
	opts := &Options{Source: "   "}
	require.NoError(t, normalizeImportSource(opts))
	assert.Equal(t, "internal", opts.Source)
}

func TestNormalizeImportSource_AllowsExternalAndNormalizes(t *testing.T) {
	opts := &Options{Source: " EXTERNAL "}
	require.NoError(t, normalizeImportSource(opts))
	assert.Equal(t, "external", opts.Source)
}

func TestNormalizeImportSource_RejectsInvalidValue(t *testing.T) {
	opts := &Options{Source: "partner"}
	err := normalizeImportSource(opts)
	require.Error(t, err)
	assert.True(t, IsValidationError(err))
	assert.Contains(t, err.Error(), "source must be internal or external")
}

func TestRun_RejectsInvalidSource(t *testing.T) {
	_, err := Run(context.Background(), nil, nil, Options{
		Dir:    "/tmp",
		Source: "partner",
	})
	require.Error(t, err)
	assert.True(t, IsValidationError(err))
	assert.Contains(t, err.Error(), "source must be internal or external")
}

func TestRun_RejectsMissingDir(t *testing.T) {
	_, err := Run(context.Background(), nil, nil, Options{
		Dir: "   ",
	})
	require.Error(t, err)
	assert.True(t, IsValidationError(err))
	assert.Contains(t, err.Error(), "dir is required")
}

func TestValidateSourceInstitutionPolicy_RequiresSelectorForExternalSource(t *testing.T) {
	opts := &Options{Source: "external"}
	err := validateSourceInstitutionPolicy(opts)
	require.Error(t, err)
	assert.True(t, IsValidationError(err))
	assert.Contains(t, err.Error(), "institution_id or institution_slug required when source is external")
}

func TestValidateSourceInstitutionPolicy_AllowsExternalSourceWithSelector(t *testing.T) {
	opts := &Options{Source: "external", InstitutionSlug: "hospital-bravo"}
	require.NoError(t, validateSourceInstitutionPolicy(opts))
}

func TestRun_RejectsExternalSourceWithoutCanonicalInstitutionSelector(t *testing.T) {
	_, err := Run(context.Background(), nil, nil, Options{
		Dir:    "/tmp",
		Source: "external",
	})
	require.Error(t, err)
	assert.True(t, IsValidationError(err))
	assert.Contains(t, err.Error(), "institution_id or institution_slug required when source is external")
}
