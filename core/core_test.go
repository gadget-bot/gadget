package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetupWithConfig_PopulatesClientField(t *testing.T) {
	gadget, _ := SetupWithConfig("xoxb-fake-token", "fake-secret", "", "", "", "", "3000", []string{})

	assert.NotNil(t, gadget.Client, "Expected gadget.Client to be populated after SetupWithConfig")
}

func TestGlobalAdminsFromString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"empty string", "", []string{}},
		{"single value", "U123", []string{"U123"}},
		{"multiple values", "U123,U456,U789", []string{"U123", "U456", "U789"}},
		{"values with whitespace", " U123 , U456 , U789 ", []string{"U123", "U456", "U789"}},
		{"trailing comma", "U123,U456,", []string{"U123", "U456"}},
		{"only commas", ",,", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := globalAdminsFromString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
