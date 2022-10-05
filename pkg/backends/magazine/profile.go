package magazine

import (
	"context"

	"github.com/metahub-registry/metahub-api/pkg/types"
)

func (mag Magazine) NeoInsertProfile(ctx context.Context, prf types.Profile) (prfID int64, err error) {
	prfID, success, err := mag.Neo.GetProfile(prf)
	if err != nil {
		return
	}
	if !success {
		prfID, err = mag.Neo.CreateProfile(prf)
		if err != nil {
			return
		}
	}
	return
}

func (mag Magazine) NeoLinkProfileToFilter(ctx context.Context, prfID, filterID int64) (err error) {
	return mag.Neo.LinkProfileToFilter(ctx, prfID, filterID)
}
