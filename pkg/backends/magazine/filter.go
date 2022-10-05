package magazine

import (
	"context"

	"github.com/metahub-registry/metahub-api/pkg/types"
)

func (mag Magazine) NeoInsertFilter(ctx context.Context, f types.Filter) (filterID int64, err error) {
	filterID, success, err := mag.Neo.GetFilter(f)
	if err != nil {
		return
	}
	if !success {
		filterID, err = mag.Neo.CreateFilter(f)
		if err != nil {
			return
		}
	}
	return
}
