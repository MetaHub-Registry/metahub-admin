package minio

import (
	"context"

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
		res, err = m.ManifestGetObj(ctx, up.GetObjPath())
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
	err = m.ManifestsPutObj(ctx, up.GetObjPath(), b)
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
