package config

import (
	"reflect"
	"testing"
)

func TestParseTargetNames(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]struct{}
	}{
		{
			name:     "single",
			input:    "OTSO",
			expected: map[string]struct{}{"OTSO": {}},
		},
		{
			name:     "multiple comma separated",
			input:    "OTSO,KONTIO,URHO",
			expected: map[string]struct{}{"OTSO": {}, "KONTIO": {}, "URHO": {}},
		},
		{
			name:     "with spaces and mixed case",
			input:    " Otso, KONTIO , urho ",
			expected: map[string]struct{}{"OTSO": {}, "KONTIO": {}, "URHO": {}},
		},
		{
			name:     "empty items",
			input:    "OTSO,,KONTIO",
			expected: map[string]struct{}{"OTSO": {}, "KONTIO": {}},
		},
		{
			name:     "empty string",
			input:    "",
			expected: map[string]struct{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseTargetNames(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("ParseTargetNames() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNormalizeName(t *testing.T) {
	if got := NormalizeName("  Otso  "); got != "OTSO" {
		t.Errorf("NormalizeName() = %v, want OTSO", got)
	}
}
