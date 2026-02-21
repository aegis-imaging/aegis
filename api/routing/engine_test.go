package routing

import (
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/stretchr/testify/assert"
)

func strPtr(s string) *string { return &s }

func TestMatches_WildcardRule(t *testing.T) {
	rule := &model.RoutingRule{}
	study := &model.Study{
		ProjectID: "proj-1",
		Modality:  "MRI",
		BodyPart:  "HEAD",
		Source:    "external",
	}
	assert.True(t, Matches(rule, study), "wildcard rule should match any study")
}

func TestMatches_ProjectID(t *testing.T) {
	rule := &model.RoutingRule{ProjectID: strPtr("proj-1")}

	assert.True(t, Matches(rule, &model.Study{ProjectID: "proj-1"}))
	assert.False(t, Matches(rule, &model.Study{ProjectID: "proj-2"}))
}

func TestMatches_Modality(t *testing.T) {
	rule := &model.RoutingRule{Modality: strPtr("MRI")}

	assert.True(t, Matches(rule, &model.Study{Modality: "MRI"}))
	assert.False(t, Matches(rule, &model.Study{Modality: "CT"}))
}

func TestMatches_ModalityCaseInsensitive(t *testing.T) {
	rule := &model.RoutingRule{Modality: strPtr("mri")}
	assert.True(t, Matches(rule, &model.Study{Modality: "MRI"}))

	rule2 := &model.RoutingRule{Modality: strPtr("MRI")}
	assert.True(t, Matches(rule2, &model.Study{Modality: "mri"}))
}

func TestMatches_BodyPart(t *testing.T) {
	rule := &model.RoutingRule{BodyPart: strPtr("HEAD")}

	assert.True(t, Matches(rule, &model.Study{BodyPart: "HEAD"}))
	assert.False(t, Matches(rule, &model.Study{BodyPart: "CHEST"}))
}

func TestMatches_BodyPartCaseInsensitive(t *testing.T) {
	rule := &model.RoutingRule{BodyPart: strPtr("head")}
	assert.True(t, Matches(rule, &model.Study{BodyPart: "HEAD"}))
}

func TestMatches_Source(t *testing.T) {
	rule := &model.RoutingRule{Source: strPtr("external")}

	assert.True(t, Matches(rule, &model.Study{Source: "external"}))
	assert.False(t, Matches(rule, &model.Study{Source: "internal"}))
}

func TestMatches_SourceCaseSensitive(t *testing.T) {
	// Source matching is exact (not case-insensitive like modality/bodypart)
	rule := &model.RoutingRule{Source: strPtr("external")}
	assert.False(t, Matches(rule, &model.Study{Source: "External"}))
}

func TestMatches_MultipleConditionsAllMatch(t *testing.T) {
	rule := &model.RoutingRule{
		ProjectID: strPtr("proj-1"),
		Modality:  strPtr("MRI"),
		BodyPart:  strPtr("HEAD"),
		Source:    strPtr("external"),
	}
	study := &model.Study{
		ProjectID: "proj-1",
		Modality:  "MRI",
		BodyPart:  "HEAD",
		Source:    "external",
	}
	assert.True(t, Matches(rule, study))
}

func TestMatches_MultipleConditionsOneFails(t *testing.T) {
	rule := &model.RoutingRule{
		ProjectID: strPtr("proj-1"),
		Modality:  strPtr("MRI"),
		BodyPart:  strPtr("HEAD"),
	}
	study := &model.Study{
		ProjectID: "proj-1",
		Modality:  "CT", // mismatch
		BodyPart:  "HEAD",
	}
	assert.False(t, Matches(rule, study))
}

func TestMatches_WildcardProjectSpecificModality(t *testing.T) {
	rule := &model.RoutingRule{Modality: strPtr("CT")}
	study := &model.Study{
		ProjectID: "any-project",
		Modality:  "CT",
		BodyPart:  "CHEST",
		Source:    "internal",
	}
	assert.True(t, Matches(rule, study))
}

func TestMatches_EmptyStudyModality(t *testing.T) {
	rule := &model.RoutingRule{Modality: strPtr("MRI")}
	study := &model.Study{Modality: ""}
	assert.False(t, Matches(rule, study))
}

func TestMatches_TableDriven(t *testing.T) {
	tests := []struct {
		name  string
		rule  *model.RoutingRule
		study *model.Study
		want  bool
	}{
		{
			name:  "empty rule matches empty study",
			rule:  &model.RoutingRule{},
			study: &model.Study{},
			want:  true,
		},
		{
			name:  "project only - match",
			rule:  &model.RoutingRule{ProjectID: strPtr("p1")},
			study: &model.Study{ProjectID: "p1"},
			want:  true,
		},
		{
			name:  "project only - no match",
			rule:  &model.RoutingRule{ProjectID: strPtr("p1")},
			study: &model.Study{ProjectID: "p2"},
			want:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Matches(tt.rule, tt.study))
		})
	}
}
