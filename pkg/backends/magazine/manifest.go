package magazine

import (
	"context"
	"errors"

	"github.com/opencontainers/go-digest"
	"github.com/sirupsen/logrus"
	"github.com/metahub-registry/metahub-admin/pkg/backends/minio"
	"github.com/metahub-registry/metahub-admin/pkg/upstream"
	ycfg "github.com/metahub-registry/metahub-admin/pkg/yaml/config"
	"github.com/metahub-registry/metahub-api/pkg/types"
)

func (mag Magazine) GetManifest(ctx context.Context, m ycfg.Manifest) (mani types.Manifest, err error) {
	up, err := m.ToUpstreamTriple()
	if err != nil {
		logrus.Error(err)
		return
	}
	// The Manifest cached has a Data filed with the original data blob (matching the digest)
	// in addition it has fields populated with the parsed data.
	mani, isCached, err := mag.Minio.GetManifest(ctx, up)
	if err != nil {
		logrus.Error(err)
		return
	}
	if !isCached {
		err = errors.New("manifest is not cached")
		return
	}
	mani.Upstream = types.Upstream{
		Reg:  up.Reg,
		Repo: up.Repo,
		Tag:  up.Tag,
	}
	return
}

func (mag Magazine) CachManifest(ctx context.Context, m ycfg.Manifest) (mani types.Manifest, err error) {
	up, err := m.ToUpstreamTriple()
	if err != nil {
		logrus.Error(err)
		return
	}
	// The Manifest cached has a Data filed with the original data blob (matching the digest)
	// in addition it has fields populated with the parsed data.
	mani, isCached, err := mag.Minio.GetManifest(ctx, up)
	if err != nil {
		logrus.Error(err)
		return
	}
	logrus.Infof("Caching manifest for %-101s -> %t", up.GetLabel(), isCached)
	if !isCached {
		b, blob, err := upstream.GetManifestByUpstreamTripleByTag(ctx, up)
		if err != nil {
			logrus.Error(err)
			return mani, err
		}
		rawMani, err := minio.RawManifestFromBlob(b)
		if err != nil {
			logrus.Error(err)
			return mani, err
		}
		up.Digest = rawMani.ConfigInfo().Digest
		// We need to make sure that the config-blob is cached
		cfgByte, cfgBlob, err := upstream.GetBlobByUpstreamTriple(ctx, up)
		if err != nil {
			logrus.Error(err)
			return mani, err
		}
		err = mag.Minio.PutConfig(ctx, cfgByte, cfgBlob.Digest.String())
		if err != nil {
			logrus.Error(err)
			return mani, err
		}
		// The blob 'b' is the original data blob (matching the digest), it contains refrerences to the config and layers
		mani, err = mag.Minio.NewManifestFromBytes(b)
		if err != nil {
			logrus.Error(err)
			return mani, err
		}
		up.Digest = digest.FromBytes(b)
		up.Size = blob.Size
		err = mag.Minio.PutManifest(ctx, b, up)
		if err != nil {
			logrus.Error(err)
			return mani, err
		}
		mani.Data = b
		if err != nil {
			logrus.Error(err)
			return mani, err
		}
	}
	mani.Upstream = types.Upstream{
		Reg:  up.Reg,
		Repo: up.Repo,
		Tag:  up.Tag,
	}
	return
}

func (mag Magazine) NeoInsertManifest(ctx context.Context, m types.Manifest) (maniID int64, err error) {
	if m.Digest == m.Config.Digest {
		logrus.Fatalf("NeoInsertManifest: Manifest.Digest and Manifest.Config.Digest are equal - something went wrong\n Digest:    %s\n ConfigDig: %s ", m.Digest, m.Config.Digest)
	}
	maniID, success, err := mag.Neo.GetManifest(m)
	if err != nil {
		return
	}
	if !success {
		maniID, err = mag.Neo.CreateManifest(m)
		if err != nil {
			return
		}
	}
	return
}

func (mag Magazine) NeoLinkManifestToUpstream(ctx context.Context, maniID, upID int64) (err error) {
	return mag.Neo.LinkManifestToUpstream(ctx, maniID, upID)
}

func (mag Magazine) NeoLinkManifestToConfig(ctx context.Context, maniID, cfgID int64) (err error) {
	return mag.Neo.LinkManifestToConfig(ctx, maniID, cfgID)
}

func (mag Magazine) NeoLinkManifestToLayer(ctx context.Context, maniID, lID int64, idx int) (err error) {
	return mag.Neo.LinkManifestToLayer(ctx, maniID, lID, idx)
}

func (mag Magazine) NeoLinkManifestToPlatform(ctx context.Context, maniID, platID int64) (err error) {
	return mag.Neo.LinkManifestToPlatform(ctx, maniID, platID)
}
