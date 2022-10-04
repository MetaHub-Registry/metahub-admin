package minio

import (
	"bytes"
	"context"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
	"github.com/metahub-registry/metahub-admin/pkg/upstream"
	"github.com/metahub-registry/metahub-api/pkg/types"
	"gitlab.com/qnib-golang/metahub/pkg/lib/utils"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

func (m Minio) GetLayer(ctx context.Context, up upstream.UpstreamTriple, layer types.Layer) (isCached bool, err error) {
	l := logrus.WithFields(logrus.Fields{
		"package": "backends/minio",
		"func":    "GetLayer",
		"digest":  utils.ShortenDigest(layer.Digest),
	})
	_ = l
	tr := otel.Tracer("neomin")
	ctx, span := tr.Start(ctx, "GetLayer")
	span.SetAttributes(
		attribute.Key("reg").String(up.Reg),
		attribute.Key("repo").String(up.Repo),
		attribute.Key("tag").String(up.Tag),
		attribute.Key("digest").String(utils.ShortenDigest(layer.Digest)),
	)
	defer span.End()
	_, err = m.LayerStatObj(ctx, layer.Digest)
	if err != nil {
		if err.Error() == "The specified key does not exist." {
			return false, nil
		}
		l.Error(err)
		return
	} else {
		isCached = true
	}
	return
}

func (m Minio) LayerStatObj(ctx context.Context, object string) (obj minio.ObjectInfo, err error) {
	return m.StatObj(ctx, layerBucket, object)
}

func (m Minio) LayerGetObj(ctx context.Context, path string) (layer types.Layer, err error) {
	l := logrus.WithFields(logrus.Fields{
		"package": "backends/minio",
		"func":    "LayerGetObj",
	})
	_ = l
	cli := m.BucketConfig[layerBucket].Client
	obj, err := cli.GetObject(ctx, layerBucket, path, minio.GetObjectOptions{})
	if err != nil {
		logrus.Errorf("Error getting layer: %s", err)
		return
	}
	defer obj.Close()
	b, e := io.ReadAll(obj)
	if e != nil {
		logrus.Fatal(e)
	}
	_ = b
	return
}

func (m Minio) PutLayer(ctx context.Context, b []byte, dig string) (err error) {
	l := logrus.WithFields(logrus.Fields{
		"package": "backends/minio",
		"func":    "PutLayer",
	})
	tr := otel.Tracer("neomin")
	ctx, span := tr.Start(ctx, "PutLayer")
	span.SetAttributes(
		attribute.Key("digest").String(utils.ShortenDigest(dig)),
	)
	defer span.End()
	err = m.LayerPutObj(ctx, dig, b)
	if err != nil {
		l.Error(err)
		return err
	}
	l.Infof("Put layer %s", utils.ShortenDigest(dig))
	return
}

func (m Minio) LayerPutObj(ctx context.Context, object string, data []byte) (err error) {
	cli := m.BucketConfig[layerBucket].Client
	_, err = cli.PutObject(ctx, layerBucket, object, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{})
	return
}
