package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

var ctx = context.Background()

func Test_FindManifestCollections(t *testing.T) {
	cols, err := FindManifestCollections(ctx, "../../../misc/yaml/manifests/qnib/featuretest/generic.yaml")
	assert.NoError(t, err)
	assert.Len(t, cols, 1)
	_, err = FindManifestCollections(ctx, "../../../misc/yaml/manifests/qnib/featuretest/mismatch_tag.yaml")
	assert.Error(t, err)

}
