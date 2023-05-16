package utils

import (
	"testing"

	"github.com/magiconair/properties/assert"
)

func TestVersion(t *testing.T) {
	v0, err := NewVersion("1.2.3")
	if err != nil {
		t.Fatalf("error creating version: %v", err)
	}
	assert.Equal(t, 1, v0.Major)
	assert.Equal(t, 2, v0.Minor)
	assert.Equal(t, 3, v0.Patch)
	assert.Equal(t, "1.2.3", v0.ToString())
	assert.Equal(t, `{"major":1,"minor":2,"patch":3}`, v0.ToJson())
}
