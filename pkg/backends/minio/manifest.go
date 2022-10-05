package minio

import (
	"context"
	"encoding/json"
	"io"

	"github.com/containers/image/v5/manifest"
	"github.com/minio/minio-go/v7"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/metahub-registry/metahub-admin/pkg/upstream"
	"github.com/metahub-registry/metahub-api/pkg/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

func (m Minio) GetManifest(ctx context.Context, up upstream.UpstreamTriple) (res types.Manifest, isCached bool, err error) {
	l := logrus.WithFields(logrus.Fields{
		"package": "backends/minio",
		"func":    "GetManifest",
	})
	tr := otel.Tracer("neomin")
	ctx, span := tr.Start(ctx, "GetManifest")
	span.SetAttributes(
		attribute.Key("reg").String(up.Reg),
		attribute.Key("repo").String(up.Repo),
		attribute.Key("tag").String(up.Tag),
	)
	defer span.End()
	_, err = m.ManifestsStatObj(ctx, up.GetObjPath())
	if err != nil {
		if err.Error() == "The specified key does not exist." {
			return res, false, nil
		}
		l.Error(err)
		return
	} else {
		isCached = true
		res, err = m.GetManifestViaPath(ctx, up.GetObjPath())
		if err != nil {
			l.Error(err)
			return
		}

	}
	return
}

func (m Minio) PutManifest(ctx context.Context, b []byte, up upstream.UpstreamTriple) (err error) {
	l := logrus.WithFields(logrus.Fields{
		"package": "backends/minio",
		"func":    "PutManifest",
	})
	tr := otel.Tracer("neomin")
	ctx, span := tr.Start(ctx, "PutManifestObj")
	span.SetAttributes(
		attribute.Key("reg").String(up.Reg),
		attribute.Key("repo").String(up.Repo),
		attribute.Key("tag").String(up.Tag),
	)
	defer span.End()
	err = m.ManifestsPutObj(ctx, up.GetObjPath(), up.ToBytes())
	if err != nil {
		l.Error(err)
		return err
	}
	err = m.ManifestsPutObj(ctx, up.Digest.String(), b)
	if err != nil {
		l.Error(err)
		return err
	}
	return
}

func (m Minio) NewManifestFromBytes(b []byte) (mani types.Manifest, err error) {
	l := logrus.WithFields(logrus.Fields{
		"packrusage": "backend/minio/manifest.go",
		"func":       "NewManifestFromBytes",
	})
	mt := manifest.GuessMIMEType(b)
	cntMani, err := manifest.FromBlob(b, mt)
	if err != nil {
		err = errors.WithMessagef(err, "manifest.FromBlob %s", string(b))
		l.Error(err)
	}
	bInfo := cntMani.ConfigInfo()
	mani = types.Manifest{
		MediaType: mt,
		Digest:    bInfo.Digest.String(),
		Size:      bInfo.Size,
	}
	imgInfo, err := cntMani.Inspect(m.configGetter)
	if err != nil {
		err = errors.Wrapf(err, "failed to inspect manifest")
		l.Error(err)
	}
	mani.Config = types.Config{
		Digest:    bInfo.Digest.String(),
		Size:      bInfo.Size,
		MediaType: bInfo.MediaType,
	}
	mani.Platform = types.Platform{
		Arch:    imgInfo.Architecture,
		OS:      imgInfo.Os,
		Variant: imgInfo.Variant,
	}
	for _, layer := range cntMani.LayerInfos() {
		mani.Layers = append(mani.Layers, types.Layer{
			MediaType: layer.MediaType,
			Digest:    layer.Digest.String(),
			Size:      layer.Size,
		})
	}
	return
}

func (m Minio) ManifestGetObj(ctx context.Context, path string) (mani types.Manifest, err error) {
	l := logrus.WithFields(logrus.Fields{
		"packrusage": "backend/minio/manifest.go",
		"func":       "ManifestGetObj",
	})
	cli := m.BucketConfig[manifestsBucket].Client
	obj, err := cli.GetObject(ctx, manifestsBucket, path, minio.GetObjectOptions{})
	if err != nil {
		logrus.Errorf("Error getting manifest: %s", err)
		return
	}
	defer obj.Close()
	b, e := io.ReadAll(obj)
	if e != nil {
		logrus.Fatal(e)
	}
	mani, err = m.NewManifestFromBytes(b)
	if err != nil {
		err = errors.Wrapf(err, "m.NewManifestFromBytes")
		l.Error(err)
	}
	return
}

func (m Minio) GetManifestViaPath(ctx context.Context, path string) (mani types.Manifest, err error) {
	l := logrus.WithFields(logrus.Fields{
		"packrusage": "backend/minio/manifest.go",
		"func":       "GetManifestViaPath",
	})
	cli := m.BucketConfig[manifestsBucket].Client
	obj, err := cli.GetObject(ctx, manifestsBucket, path, minio.GetObjectOptions{})
	if err != nil {
		l.Errorf("Error getting manifest via path: %s", err)
		return
	}
	defer obj.Close()
	b, e := io.ReadAll(obj)
	if e != nil {
		l.Error(e)
	}
	var up upstream.UpstreamTriple
	err = json.Unmarshal(b, &up)
	if err != nil {
		l.Error(err)
		return
	}
	return m.ManifestGetObj(ctx, up.Digest.String())
}
