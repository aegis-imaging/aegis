package model

import "testing"

func TestValidateTenantSlug(t *testing.T) {
	cases := []struct {
		slug string
		ok   bool
	}{
		// Valid.
		{"acme", true},
		{"acme-research", true},
		{"abc", true},
		{"a1b2c3", true},
		{"twelve-char-x", true},

		// Invalid.
		{"", false},
		{"AB", false},                 // too short and uppercase
		{"acme_research", false},      // underscore not allowed
		{"-leading-hyphen", false},    // leading hyphen
		{"trailing-hyphen-", false},   // trailing hyphen
		{"Acme", false},               // uppercase
		{"acme.research", false},      // dot
		{"acme research", false},      // space
		{"a", false},                  // too short (1 char)
		{"ab", false},                 // too short (2 chars)
		{"this-slug-is-way-too-long-it-exceeds-the-64-char-limit-by-quite-a-lot", false},
	}
	for _, c := range cases {
		err := ValidateTenantSlug(c.slug)
		if c.ok && err != nil {
			t.Errorf("ValidateTenantSlug(%q) unexpectedly rejected: %v", c.slug, err)
		}
		if !c.ok && err == nil {
			t.Errorf("ValidateTenantSlug(%q) should have been rejected", c.slug)
		}
	}
}
