package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetupWithConfig_PopulatesClientField(t *testing.T) {
	gadget, _ := SetupWithConfig("xoxb-fake-token", "fake-secret", "", "", "", "", "3000", []string{})

	assert.NotNil(t, gadget.Client, "Expected gadget.Client to be populated after SetupWithConfig")
}
