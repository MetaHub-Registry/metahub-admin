package sqlite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/metahub-registry/metahub-core/pkg/types"
)

var (
	testFilterGeneric = types.Filter{
		Label:   "generic",
		Feature: "",
	}
)

func Test_Filter_Insert(t *testing.T) {
	_, found, err := sq.Filter.GetFilter(testFilterGeneric)
	assert.NoError(t, err)
	assert.False(t, found)
	err = sq.Filter.InsertFilter(&testFilterGeneric)
	assert.NoError(t, err)
	assert.NotEqual(t, int64(-1), testCfg.SqlID)
	id, found, err := sq.Filter.GetFilter(testFilterGeneric)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, testFilterGeneric.SqlID, id)
}
