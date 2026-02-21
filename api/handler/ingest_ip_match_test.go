package handler

import (
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectBestInstitutionFromIPMatches_NoMatches(t *testing.T) {
	inst, err := selectBestInstitutionFromIPMatches(nil)
	require.NoError(t, err)
	assert.Nil(t, inst)
}

func TestSelectBestInstitutionFromIPMatches_PrefersMostSpecificPrefix(t *testing.T) {
	instA := &model.Institution{ID: "inst-a"}
	instB := &model.Institution{ID: "inst-b"}

	best, err := selectBestInstitutionFromIPMatches([]ipInstitutionMatch{
		{Institution: instA, PrefixLen: 16},
		{Institution: instB, PrefixLen: 24},
	})
	require.NoError(t, err)
	require.NotNil(t, best)
	assert.Equal(t, "inst-b", best.ID)
}

func TestSelectBestInstitutionFromIPMatches_AmbiguousSamePrefix(t *testing.T) {
	instA := &model.Institution{ID: "inst-a"}
	instB := &model.Institution{ID: "inst-b"}

	best, err := selectBestInstitutionFromIPMatches([]ipInstitutionMatch{
		{Institution: instA, PrefixLen: 16},
		{Institution: instB, PrefixLen: 16},
	})
	require.Error(t, err)
	assert.Nil(t, best)
}

func TestSelectBestInstitutionFromIPMatches_DuplicateInstitutionNotAmbiguous(t *testing.T) {
	instA := &model.Institution{ID: "inst-a"}

	best, err := selectBestInstitutionFromIPMatches([]ipInstitutionMatch{
		{Institution: instA, PrefixLen: 16},
		{Institution: instA, PrefixLen: 16},
	})
	require.NoError(t, err)
	require.NotNil(t, best)
	assert.Equal(t, "inst-a", best.ID)
}

func TestParseIPRange_SingleIP(t *testing.T) {
	network := parseIPRange("10.1.2.3")
	require.NotNil(t, network)
	assert.Equal(t, "10.1.2.3/32", network.String())
}

func TestParseIPRange_CIDR(t *testing.T) {
	network := parseIPRange("10.1.0.0/16")
	require.NotNil(t, network)
	assert.Equal(t, "10.1.0.0/16", network.String())
}

func TestParseIPRange_Invalid(t *testing.T) {
	network := parseIPRange("not-an-ip")
	assert.Nil(t, network)
}
