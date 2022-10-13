package upstream

import (
	"bytes"
	"context"
	"fmt"

	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"github.com/regclient/regclient"
	"github.com/regclient/regclient/types"
	"github.com/regclient/regclient/types/ref"
	"github.com/sirupsen/logrus"
	mhTypes "github.com/metahub-registry/metahub-api/pkg/types"
	"gitlab.com/qnib-golang/metahub/pkg/lib/utils"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

func GetBlobByUpstreamTriple(ctx context.Context, up UpstreamTriple) (b []byte, blob mhTypes.BlobMeta, err error) {
	shortDig := utils.ShortenDigest(up.Digest.String())
	l := logrus.WithFields(logrus.Fields{
		"package":    "neomin",
		"func":       "GetBlobByUpstreamTriple",
		"repository": up.Repo,
		"registry":   up.Reg,
		"digest":     shortDig,
	})
	if up.Digest.String() == "" {
		l.Fatal("No digest provided")
	}
	tr := otel.Tracer("neomin")
	blobUrl := fmt.Sprintf("%s/%s", up.Reg, up.Repo)
	ctx, span := tr.Start(ctx, "GetBlobByUpstreamTriple")
	span.SetAttributes(attribute.Key("url").String(blobUrl))
	span.SetAttributes(attribute.Key("digest").String(shortDig))
	defer span.End()
	blob = mhTypes.BlobMeta{}
	rc := regclient.New()

	ref, err := ref.New(blobUrl)
	if err != nil {
		l.Errorf("Failed creating ref: %v", err)
	}
	ref.Digest = up.Digest.String()
	desc := types.Descriptor{Digest: up.Digest, Size: up.Size}
	blobReader, err := rc.BlobGet(ctx, ref, desc)
	if err != nil {
		err = errors.Wrap(err, "while rc.BlogGet with digest")
		l.Fatalf(err.Error())
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(blobReader)
	b = buf.Bytes()
	blob.Size = int64(len(b))
	blob.Digest = up.Digest
	blob.MediaType = up.MediaType
	return
}

func GetBlobByUpstreamTripleByDigest(ctx context.Context, up UpstreamTriple, dig string) (b []byte, blob mhTypes.BlobMeta, err error) {
	l := logrus.WithFields(logrus.Fields{
		"package":    "neomin",
		"func":       "GetBlobByUpstreamTripleByDigest",
		"repository": up.Repo,
		"registry":   up.Reg,
		"digest":     utils.ShortenDigest(dig),
	})
	if dig == "" {
		logrus.Fatal("No digest provided")
	}
	tr := otel.Tracer("neomin")
	blobUrl := fmt.Sprintf("%s/%s", up.Reg, up.Repo)
	ctx, span := tr.Start(ctx, "GetBlobByUpstreamTripleByDigest")
	span.SetAttributes(attribute.Key("url").String(blobUrl))
	span.SetAttributes(attribute.Key("digest").String(utils.ShortenDigest(dig)))
	defer span.End()
	blob = mhTypes.BlobMeta{}
	rc := regclient.New()

	ref, err := ref.New(blobUrl)
	if err != nil {
		l.Errorf("Failed creating ref: %v", err)
	}
	ref.Digest = dig
	d := digest.Digest(dig)
	desc := types.Descriptor{Digest: d, Size: up.Size}
	blobReader, err := rc.BlobGet(ctx, ref, desc)
	if err != nil {
		err = errors.Wrapf(err, "while rc.BlogGet with digest: %s", dig)
		l.Fatalf(err.Error())
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(blobReader)
	b = buf.Bytes()
	blob.Size = int64(len(b))
	blob.Digest = d
	return
}

func GetManifestByUpstreamTripleByTag(ctx context.Context, up UpstreamTriple) (b []byte, blob mhTypes.BlobMeta, err error) {
	l := logrus.WithFields(logrus.Fields{
		"package":    "neomin",
		"func":       "GetBlobByUpstreamTriple",
		"repository": up.Repo,
		"tag":        up.Tag,
		"registry":   up.Reg,
	})
	tr := otel.Tracer("neomin")
	blobUrl := fmt.Sprintf("%s/%s:%s", up.Reg, up.Repo, up.Tag)
	logrus.Info("blobUrl: ", blobUrl)
	ctx, span := tr.Start(ctx, "GetBlobByUpstreamTriple")
	span.SetAttributes(attribute.Key("url").String(blobUrl))
	defer span.End()
	blob = mhTypes.BlobMeta{}
	rc := regclient.New()

	ref, err := ref.New(blobUrl)
	if err != nil {
		l.Errorf("Failed creating ref: %v", err)
	}
	mani, err := rc.ManifestHead(ctx, ref)
	if err != nil {
		l.Fatalf(err.Error())
	}
	ref.Digest = mani.GetDescriptor().Digest.String()
	blob.Digest = mani.GetDescriptor().Digest
	blob.Size = mani.GetDescriptor().Size
	blob.MediaType = mani.GetDescriptor().MediaType
	mani, err = rc.ManifestGet(ctx, ref)
	b, err = mani.RawBody()
	if err != nil {
		l.Fatalf(err.Error())
	}
	return
}
