package magazine

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/metahub-registry/metahub-admin/pkg/upstream"
	"github.com/metahub-registry/metahub-api/pkg/types"
	"gitlab.com/qnib-golang/metahub/pkg/lib/utils"
)

func (mag Magazine) CacheLayers(ctx context.Context, mani types.Manifest) (err error) {
	up := upstream.UpstreamTriple{
		Reg:  mani.Upstream.Reg,
		Repo: mani.Upstream.Repo,
		Tag:  mani.Upstream.Tag,
	}
	for idx, layer := range mani.Layers {
		isCached, e := mag.Minio.GetLayer(ctx, up, layer)
		if e != nil {
			err = e
			logrus.Error(err)
			return
		}
		logrus.Infof("Caching layer[%-2d] %s for %-90s -> %t", idx, utils.ShortenDigest(layer.Digest), up.GetLabel(), isCached)
		if !isCached {
			b, blobs, err := upstream.GetBlobByUpstreamTripleByDigest(ctx, up, layer.Digest)
			if err != nil {
				err = errors.Wrapf(err, "failed to get blob for %s", up.GetLabel())
				logrus.Error(err)
				return err
			}
			_ = blobs
			err = mag.Minio.PutLayer(ctx, b, layer.Digest)
			if err != nil {
				logrus.Error(err)
				return err
			}
		}
	}
	return
}
