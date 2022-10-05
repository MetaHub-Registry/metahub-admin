package minio

import (
	"testing"

	"github.com/containers/image/v5/types"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
)

func Test_ConfigGetter(t *testing.T) {
	bi := types.BlobInfo{
		Digest: digest.Digest(featureTestGenericCfgDigest),
	}
	cfg, err := min.configGetter(bi)
	assert.NoError(t, err)
	assert.Equal(t, featureTestGenericCfg, cfg)
}
