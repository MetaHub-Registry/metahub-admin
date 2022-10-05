package magazine

import (
	"context"
	"encoding/json"

	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/metahub-registry/metahub-admin/pkg/upstream"
	"github.com/metahub-registry/metahub-api/pkg/types"
)

func (mag Magazine) CachImageConfig(ctx context.Context, mani *types.Manifest) (err error) {
	up := upstream.UpstreamTriple{
		Reg:    mani.Upstream.Reg,
		Repo:   mani.Upstream.Repo,
		Tag:    mani.Upstream.Tag,
		Digest: digest.FromString(mani.Config.Digest),
	}

	cfg, isCached, err := mag.Minio.GetConfig(ctx, up, mani.Config)
	_ = cfg
	if err != nil {
		logrus.Error(err)
		return
	}
	logrus.Infof("Caching config for %-103s -> %t", up.GetLabel(), isCached)
	if !isCached {
		b, blob, err := upstream.GetBlobByUpstreamTripleByDigest(ctx, up, mani.Config.Digest)
		if err != nil {
			err = errors.Wrapf(err, "failed to get blob for %s", up.GetLabel())
			logrus.Error(err)
			return err
		}

		logrus.Warnf("Blob fetched: %v", blob)
		err = mag.Minio.PutConfig(ctx, b, mani.Config.Digest)
		if err != nil {
			logrus.Error(err)
			return err
		}
		// unmarshal config
		var imgCfg v1.Image
		err = json.Unmarshal(b, &imgCfg)
		if err != nil {
			logrus.Error(err)
			return err
		}
		mani.Platform.Arch = imgCfg.Architecture
		mani.Platform.OS = imgCfg.OS
		mani.Platform.Variant = imgCfg.Variant
	}

	return
}

func (mag Magazine) NeoInsertConfig(ctx context.Context, cfg types.Config, up types.Upstream) (cfgID int64, err error) {
	cfgID, success, err := mag.Neo.GetConfig(cfg)
	if err != nil {
		return
	}
	if !success {
		cfgID, err = mag.Neo.CreateConfig(cfg, up)
		if err != nil {
			return
		}
	}
	return
}
