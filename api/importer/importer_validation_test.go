package importer

import (
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
