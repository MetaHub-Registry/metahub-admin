package minio

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ManifestFromBlob(t *testing.T) {
	mani, err := min.NewManifestFromBytes(featureTestGenericMani)
	assert.NoError(t, err)
	assert.Equal(t, "application/vnd.docker.distribution.manifest.v2+json", mani.MediaType)
	assert.Equal(t, 2, len(mani.Layers))
	assert.Equal(t, "amd64", mani.Platform.Arch)
	assert.Equal(t, "linux", mani.Platform.OS)
}
