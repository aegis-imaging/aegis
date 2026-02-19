package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Mayo Clinic", "mayo-clinic"},
		{"  Leading Spaces  ", "leading-spaces"},
		{"Special!@#Characters", "specialcharacters"},
		{"Under_scores", "under-scores"},
		{"already-slug", "already-slug"},
		{"UPPERCASE", "uppercase"},
		{"123 Numbers", "123-numbers"},
		{"", ""},
		{"---dashes---", "dashes"},
		{"Multiple   Spaces", "multiple---spaces"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, slugify(tt.input))
		})
	}
}
