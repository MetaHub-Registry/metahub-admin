/*
Copyright © 2026 Christian Kniep <christian@qnib.org>
*/
package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/opencontainers/go-digest"
	"github.com/regclient/regclient"
	rcConfig "github.com/regclient/regclient/config"
	rcTypes "github.com/regclient/regclient/types"
	"github.com/regclient/regclient/types/manifest"
	"github.com/regclient/regclient/types/ref"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/metahub-registry/metahub-admin/pkg/config"
	"github.com/metahub-registry/metahub-core/pkg/backends/magazine"
	"github.com/metahub-registry/metahub-core/pkg/types"
	yamlCfg "github.com/metahub-registry/metahub-core/pkg/yaml/config"
)

var (
	rpArtifactRepo string
	rpArtifactType string
	rpName         string
	rpQuantTag     string
	rpProfile      string
	rpRuntime      string
	rpTargetPath   string
	rpRegistryURL  string
	rpRegistryUser string
	rpRegistryPass string
	rpRegistryTLS  bool
	rpSkipManip    bool
	rpConfigFile   string
)

// repackageCmd turns an ORAS model artifact into an injectable OCI model-layer
// image (one uncompressed tar layer per model file), pushes it to metahub itself
// so metahub's push handlers register the Manifest/Blob graph nodes, and upserts
// a `model/<name>` manipulation (vram:<tier> variant) directly into Neo4j.
var repackageCmd = &cobra.Command{
	Use:   "repackage",
	Short: "Repackage an ORAS model artifact into an injectable OCI model-layer image",
	Long: `Reads an ORAS model artifact (one or more same-quantization model files),
wraps each file into a deterministic uncompressed tar layer at /model/<name>/<file>,
pushes the assembled image to metahub (blob -> config -> manifest) so it becomes a
resolvable image, and persists a model/<name> manipulation to Neo4j attached to
--profile so a pull under that profile injects it (no new pull-path logic).`,
	Run: repackageRun,
}

func init() {
	rootCmd.AddCommand(repackageCmd)
	f := repackageCmd.Flags()
	f.StringVar(&rpArtifactRepo, "artifact-repo", "", "ORAS artifact repo (e.g. models/qwen3.5-122b)")
	f.StringVar(&rpArtifactType, "artifact-type", "", "ORAS artifactType to look up")
	f.StringVar(&rpName, "name", "", "model name (manipulation model/<name>, in-image /model/<name>/)")
	f.StringVar(&rpQuantTag, "quant-tag", "", "quantization tag → model-layer image tag")
	f.StringVar(&rpProfile, "profile", "", "profile to attach the model manipulation to (the profile the model is pulled under)")
	f.StringVar(&rpRuntime, "runtime", "", "runtime (application container) the config composes with; recorded as a config-artefact annotation for composition discovery")
	f.StringVar(&rpTargetPath, "target-path", "", "in-image path (default /model/<name>/)")
	f.StringVar(&rpRegistryURL, "registry-url", "", "metahub OCI registry host to push to (e.g. metahub.ns.svc)")
	f.StringVar(&rpRegistryUser, "registry-user", "", "registry push user")
	f.StringVar(&rpRegistryPass, "registry-pass", "", "registry push password/token")
	f.BoolVar(&rpRegistryTLS, "registry-tls", true, "use TLS for the registry push")
	f.BoolVar(&rpSkipManip, "skip-manipulation", false, "push the model-image but do not create the model/<name> manipulation (image-production only)")
	f.StringVar(&rpConfigFile, "config", "", "config file to attach as the model config; when --runtime is set and the source ORAS artifact has no config blob, a config artefact is synthesised (from this file, or a minimal default) so the model is servable")
}

func repackageRun(cmd *cobra.Command, args []string) {
	config.SetCommandValues(cmd)
	mag, ctx, err := magazine.NewMagazine(cmd, "")
	if err != nil {
		logrus.Fatal(err)
	}
	repackageOnce(ctx, mag)
}

// repackageOnce runs the full repackage flow for the current rp* parameters
// (artifact-repo/type, name, quant-tag, profile, runtime, registry-*). It is the
// reusable core shared by the `repackage` CLI and the declarative `parse models`
// orchestrator, which sets the rp* vars per (quant, profile) and calls it.
func repackageOnce(ctx context.Context, mag magazine.Magazine) {
	// 1. read the ORAS artifact + plan the per-file layers (ORAS order, reject mixed-quant)
	art, err := mag.Database.OrasArtifactGetByName(ctx, rpArtifactRepo, rpArtifactType)
	if err != nil {
		logrus.Fatalf("OrasArtifactGetByName(%s,%s): %v", rpArtifactRepo, rpArtifactType, err)
	}
	plans, err := types.PlanModelRepackage(art.Blobs, rpName, rpTargetPath)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Infof("repackage: %d model file(s) for %q → models/%s-layer:%s", len(plans), rpName, rpName, rpQuantTag)

	// 2. regclient (auth) to push to metahub; image ref = <reg>/models/<name>-layer:<tag>
	rc := newPushClient(rpRegistryURL, rpRegistryUser, rpRegistryPass, rpRegistryTLS)
	defer rc.Close(ctx, ref.Ref{})
	imageRefStr := fmt.Sprintf("%s/models/%s-layer:%s", rpRegistryURL, rpName, rpQuantTag)
	imgRef, err := ref.New(imageRefStr)
	if err != nil {
		logrus.Fatalf("ref.New(%s): %v", imageRefStr, err)
	}

	// 3–4. push each model file as a tar layer, then assemble + push the
	//      model-layer OCI image (subject-linked to the source ORAS artefact).
	maniDig, modelManiSize := pushModelLayerImage(ctx, mag, rc, imgRef, plans, art.Digest, art.MediaType)

	if rpSkipManip {
		logrus.Infof("--skip-manipulation set: model-image pushed, not creating manipulation model/%s", rpName)
		return
	}

	// 5. Wire Upstream+Manifest DIRECTLY from the push-registered manifest so the
	//    manipulation engine can resolve the pushed image (no re-fetch/auth; metahub
	//    already registered the Manifest node on push).
	if err := wireImageUpstream(ctx, mag, imageRefStr, maniDig); err != nil {
		logrus.Fatal(err)
	}

	// 6–7. Register the quant as a variant of the model/<name> manipulation
	//      (idempotent, additive) and attach it to the pushed-under profile, so a
	//      pull under that profile injects it.
	prf := linkModelToProfile(ctx, mag, maniDig)

	// 8. Config injection (optional): a configuration blob in the same artifact
	//    (identified by ModelConfigMediaType) becomes a per-profile config layer +
	//    a sibling config/<name> manipulation attached to the same profile, so a
	//    pull under it injects the config alongside the model.
	if cfgPlans := types.PlanConfigRepackage(art.Blobs, rpName); len(cfgPlans) > 0 {
		if e := repackageConfig(ctx, mag, rc, cfgPlans, prf.ID, maniDig.String(), modelManiSize); e != nil {
			logrus.Fatalf("config repackage: %v", e)
		}
	} else if rpRuntime != "" {
		// A gguf-only source has no config blob, but a runtime binding was requested.
		// The runtime annotation lives on the config artefact, and the model-centric
		// serve path resolves the runtime from it — so synthesise a config artefact
		// (default or --config content) carrying {profile, quant, runtime}. Without
		// this the model has no composition and is not servable under models/<name>.
		if e := repackageSyntheticConfig(ctx, mag, rc, prf.ID, maniDig.String(), modelManiSize, defaultConfigBytes()); e != nil {
			logrus.Fatalf("synthetic config repackage: %v", e)
		}
	} else {
		logrus.Infof("no config blob (%s) and no --runtime — model-only push", types.ModelConfigMediaType)
	}

	// 9. Warm the manipulation cache for THIS serving profile so the first pull is
	//    a cache hit instead of the on-request createManipulatedManifest recompute.
	warmManipulationCache(ctx, mag, prf)
}

// warmManipulationCache precomputes + persists the model manipulation for prf so
// a subsequent model-centric pull is a cache hit rather than paying the ~1.7s
// createManipulatedManifest recompute on the request path (against eu-west-1 S3).
// The model-centric pull resolves models/<name> to the runtime collection at tag
// "latest" and manipulates it under the pulling profile; this runs that exact
// manipulation now (populating reuseExistingManipulated). Done via
// ApplyManipulationsToManifests directly — not ComputeManifestlistsForRepoTag —
// to avoid broadcasting the profile's annotations onto the tag. Non-fatal: on any
// miss the pull falls back to the lazy compute as before.
func warmManipulationCache(ctx context.Context, mag magazine.Magazine, prf types.Profile) {
	if rpRuntime == "" {
		return
	}
	warmReq := types.Request{Repo: rpRuntime, Tag: "latest", Profile: prf.Label}
	ml, found, err := mag.IndexGetByRequest(ctx, warmReq)
	if err != nil {
		logrus.Warnf("precompute: IndexGetByRequest(%s:latest, %s) failed (non-fatal): %v", rpRuntime, prf.Label, err)
		return
	}
	if !found || len(ml.Manifests) == 0 {
		logrus.Infof("precompute: no runtime index for %s:latest (profile %s) yet — pull will compute lazily", rpRuntime, prf.Label)
		return
	}
	if _, mErr := mag.ApplyManipulationsToManifests(ctx, prf, ml.Manifests); mErr != nil {
		logrus.Warnf("precompute: ApplyManipulationsToManifests(%s, %s:latest) failed (non-fatal): %v", prf.Label, rpRuntime, mErr)
		return
	}
	logrus.Infof("precomputed manipulation for profile %s on %s:latest (%d manifests) — first pull is a cache hit", prf.Label, rpRuntime, len(ml.Manifests))
}

// repackageConfig builds a `models/<name>-config:<quant>` image from the config
// layer plan(s), wires it resolvable (no self-refetch), and attaches a sibling
// `config/<name>` manipulation to the profile — the config analogue of steps 3–7
// for the model, so a pull under the profile injects the config alongside it.
func repackageConfig(ctx context.Context, mag magazine.Magazine, rc *regclient.RegClient, plans []types.ModelFilePlan, prfID int64, modelManiDigest string, modelManiSize int64) error {
	imageRefStr := fmt.Sprintf("%s/models/%s-config:%s", rpRegistryURL, rpName, rpQuantTag)
	imgRef, err := ref.New(imageRefStr)
	if err != nil {
		return fmt.Errorf("ref.New(%s): %w", imageRefStr, err)
	}
	layers, err := pushLayerPlans(ctx, mag, rc, imgRef, plans)
	if err != nil {
		return err
	}
	// Decision A: the config is a sibling subject-linked artefact. Its OCI subject
	// points at the model-layer image manifest (so referrers(model) yields it) and
	// its annotations {profile, runtime, quant} identify the composition point.
	annotations := types.ConfigCompositionAnnotations(rpProfile, rpQuantTag, rpRuntime)
	cfg, cfgDig, mani, maniDig, err := types.AssembleModelImage("", "", layers, modelManiDigest, rcTypes.MediaTypeOCI1Manifest, modelManiSize, annotations)
	if err != nil {
		return err
	}
	if err := pushAssembledImage(ctx, rc, imgRef, cfg, cfgDig, mani, maniDig); err != nil {
		return err
	}
	logrus.Infof("pushed config-image models/%s-config:%s (manifest %s)", rpName, rpQuantTag, maniDig)
	if err := wireImageUpstream(ctx, mag, imageRefStr, maniDig); err != nil {
		return err
	}

	return createConfigManipulation(ctx, mag, imageRefStr, prfID)
}

// createConfigManipulation attaches a sibling config/<name> manipulation to the
// profile: the config-image layer plus a QNIB_MODEL_CONFIG_DIR env var pointing a
// stock base entrypoint at the injected config directory (opt-in contract).
func createConfigManipulation(ctx context.Context, mag magazine.Magazine, imageRefStr string, prfID int64) error {
	configDir := "/model/" + rpName + "/config"
	cfgMp := yamlCfg.Manipulation{
		Label:        yamlCfg.ConfigManipulationLabel(rpName),
		ManipuLayers: []yamlCfg.ManipuLayer{{Image: imageRefStr}},
		Env:          map[string]string{"QNIB_MODEL_CONFIG_DIR": configDir},
	}
	mpc, err := mag.Database.ManipulationFromYAML(ctx, cfgMp)
	if err != nil {
		return err
	}
	mpcID, err := mag.ManipulationGetCreate(ctx, mpc)
	if err != nil {
		return err
	}
	if err := mag.Database.ProfileLinkToManipulation(ctx, prfID, mpcID); err != nil {
		return fmt.Errorf("ProfileLinkToManipulation(config, prf=%d, mp=%d): %w", prfID, mpcID, err)
	}
	logrus.Infof("config manipulation %s (id %d) linked to profile id %d", mpc.Label, mpcID, prfID)
	return nil
}

// defaultConfigBytes returns the config content for a synthesised config artefact:
// the --config file when provided, else a minimal default. The full B5 render
// (abstract keys → per-runtime template) is future work; this is enough to carry the
// runtime binding + a usable default config so the model is servable.
func defaultConfigBytes() []byte {
	if rpConfigFile != "" {
		b, err := os.ReadFile(rpConfigFile)
		if err != nil {
			logrus.Fatalf("read --config %s: %v", rpConfigFile, err)
		}
		return b
	}
	return []byte("{\"ctx_size\":32768,\"kv_cache_type\":\"q8_0\"}\n")
}

// repackageSyntheticConfig builds a config artefact for a model whose source ORAS
// artifact carries no config blob (e.g. a gguf-only model). It streams configBytes
// into an uncompressed tar layer at model/<name>/config/config.json, pushes it, and
// assembles a sibling subject-linked config artefact carrying {profile, quant,
// runtime} — the composition point the model-centric serve path resolves the runtime
// from. This is the config analogue of the model push for the no-source-config case.
func repackageSyntheticConfig(ctx context.Context, mag magazine.Magazine, rc *regclient.RegClient, prfID int64, modelManiDigest string, modelManiSize int64, configBytes []byte) error {
	imageRefStr := fmt.Sprintf("%s/models/%s-config:%s", rpRegistryURL, rpName, rpQuantTag)
	imgRef, err := ref.New(imageRefStr)
	if err != nil {
		return fmt.Errorf("ref.New(%s): %w", imageRefStr, err)
	}
	entry := types.ModelConfigTarget(rpName, "config.json")
	var buf bytes.Buffer
	dig, size, err := types.StreamModelLayer(&buf, bytes.NewReader(configBytes), int64(len(configBytes)), entry)
	if err != nil {
		return fmt.Errorf("stream synthetic config layer: %w", err)
	}
	if _, err := rc.BlobPut(ctx, imgRef, rcTypes.Descriptor{
		MediaType: types.ModelLayerMediaType, Digest: dig, Size: size,
	}, &buf); err != nil {
		return fmt.Errorf("push synthetic config layer: %w", err)
	}
	layers := []types.ModelLayerDescriptor{{Digest: dig, Size: size}}
	annotations := types.ConfigCompositionAnnotations(rpProfile, rpQuantTag, rpRuntime)
	cfg, cfgDig, mani, maniDig, err := types.AssembleModelImage("", "", layers, modelManiDigest, rcTypes.MediaTypeOCI1Manifest, modelManiSize, annotations)
	if err != nil {
		return err
	}
	if err := pushAssembledImage(ctx, rc, imgRef, cfg, cfgDig, mani, maniDig); err != nil {
		return err
	}
	logrus.Infof("pushed synthetic config-image models/%s-config:%s (manifest %s, runtime %s)", rpName, rpQuantTag, maniDig, rpRuntime)
	if err := wireImageUpstream(ctx, mag, imageRefStr, maniDig); err != nil {
		return err
	}
	return createConfigManipulation(ctx, mag, imageRefStr, prfID)
}

// newPushClient builds an authenticated regclient for pushing to metahub.
func newPushClient(reg, user, pass string, tls bool) *regclient.RegClient {
	host := rcConfig.Host{Name: reg, Hostname: reg, User: user, Pass: pass, TLS: rcConfig.TLSEnabled}
	if !tls {
		host.TLS = rcConfig.TLSDisabled
	}
	return regclient.New(regclient.WithConfigHost(host))
}

// hashModelLayer streams the source blob through the deterministic tar to
// io.Discard to obtain the layer digest (== DiffID, uncompressed) and tar size,
// at constant memory. Pass 1 of the push.
func hashModelLayer(ctx context.Context, mag magazine.Magazine, p types.ModelFilePlan) (digest.Digest, int64, error) {
	src, err := mag.Minio.BlobGetReader(ctx, p.Blob.Digest)
	if err != nil {
		return "", 0, err
	}
	defer src.Close()
	return types.StreamModelLayer(io.Discard, src, p.Blob.Size, p.Entry)
}

// pushModelLayer streams the source blob through the deterministic tar straight
// into BlobPut (via an io.Pipe), so the layer is uploaded to metahub at constant
// memory. Pass 2 of the push (dig/size known from pass 1).
func pushModelLayer(ctx context.Context, mag magazine.Magazine, rc *regclient.RegClient, imgRef ref.Ref, p types.ModelFilePlan, dig digest.Digest, size int64) error {
	src, err := mag.Minio.BlobGetReader(ctx, p.Blob.Digest)
	if err != nil {
		return err
	}
	defer src.Close()
	pr, pw := io.Pipe()
	go func() {
		_, _, e := types.StreamModelLayer(pw, src, p.Blob.Size, p.Entry)
		pw.CloseWithError(e)
	}()
	_, err = rc.BlobPut(ctx, imgRef, rcTypes.Descriptor{
		MediaType: types.ModelLayerMediaType, Digest: dig, Size: size,
	}, pr)
	return err
}

// pushLayerPlans pushes each planned model file as a deterministic uncompressed
// tar layer, returning the layer descriptors. The tar is content-addressed and
// deterministic, so a layer already present in the target repo (from a prior
// seed/admin build, or an earlier profile in the same seed) is HEAD-detected and
// its slow (~2-4 MB/s S3-egress) upload skipped. The hash pass still runs — it's
// a fast S3 read — to obtain the digest to check against.
func pushLayerPlans(ctx context.Context, mag magazine.Magazine, rc *regclient.RegClient, imgRef ref.Ref, plans []types.ModelFilePlan) ([]types.ModelLayerDescriptor, error) {
	layers := make([]types.ModelLayerDescriptor, 0, len(plans))
	for _, p := range plans {
		// INSTRUMENTATION (perf-debug): time the two S3-bound passes separately —
		// hashModelLayer reads the source blob (S3 ingress) to derive the tar digest;
		// pushModelLayer streams the tar back (S3 egress). Logging both, plus the HEAD
		// dedup result, shows exactly where each profile's ~1GB of wall-time goes.
		hStart := time.Now()
		dig, tarSize, err := hashModelLayer(ctx, mag, p)
		if err != nil {
			return nil, fmt.Errorf("hash layer %s: %w", p.File, err)
		}
		logrus.Infof("repackage-instr: hashed %s → %s (%d bytes) read+hash=%s", p.File, dig, tarSize, time.Since(hStart))
		present := blobPresent(ctx, rc, imgRef, dig, tarSize)
		logrus.Infof("repackage-instr: HEAD dedup blobPresent(repo=%s digest=%s)=%v", imgRef.Repository, dig, present)
		if present {
			layers = append(layers, types.ModelLayerDescriptor{Digest: dig, Size: tarSize})
			logrus.Infof("layer %s → %s already present (%d bytes) — skipping upload", p.File, dig, tarSize)
			continue
		}
		pStart := time.Now()
		if err := pushModelLayer(ctx, mag, rc, imgRef, p, dig, tarSize); err != nil {
			return nil, fmt.Errorf("push layer %s: %w", p.File, err)
		}
		layers = append(layers, types.ModelLayerDescriptor{Digest: dig, Size: tarSize})
		logrus.Infof("pushed layer %s → %s (%d bytes) upload=%s", p.File, dig, tarSize, time.Since(pStart))
	}
	return layers, nil
}

// blobPresent reports whether a blob with digest dig already exists in imgRef's
// repo, so the slow S3-egress layer upload can be skipped. BlobHead returns a
// non-nil error on a HEAD miss (or any transport error); either way we treat it
// as "not present" and fall back to a full push, which is always safe.
func blobPresent(ctx context.Context, rc *regclient.RegClient, imgRef ref.Ref, dig digest.Digest, size int64) bool {
	_, err := rc.BlobHead(ctx, imgRef, rcTypes.Descriptor{Digest: dig, Size: size})
	return err == nil
}

// pushAssembledImage pushes the assembled OCI config blob then the manifest.
func pushAssembledImage(ctx context.Context, rc *regclient.RegClient, imgRef ref.Ref, cfg []byte, cfgDig digest.Digest, mani []byte, maniDig digest.Digest) error {
	if _, err := rc.BlobPut(ctx, imgRef, rcTypes.Descriptor{
		MediaType: rcTypes.MediaTypeOCI1ImageConfig, Digest: cfgDig, Size: int64(len(cfg)),
	}, bytes.NewReader(cfg)); err != nil {
		return fmt.Errorf("push config blob: %w", err)
	}
	m, err := manifest.New(
		manifest.WithRaw(mani),
		manifest.WithDesc(rcTypes.Descriptor{
			MediaType: rcTypes.MediaTypeOCI1Manifest, Digest: maniDig, Size: int64(len(mani)),
		}),
	)
	if err != nil {
		return fmt.Errorf("manifest.New: %w", err)
	}
	if err := rc.ManifestPut(ctx, imgRef, m); err != nil {
		return fmt.Errorf("push manifest: %w", err)
	}
	return nil
}

// pushModelLayerImage runs steps 3–4: push the model files as tar layers, then
// assemble + push the model-layer OCI image with a subject reference to the source
// artefact. Returns the pushed manifest digest and its byte size.
func pushModelLayerImage(ctx context.Context, mag magazine.Magazine, rc *regclient.RegClient, imgRef ref.Ref, plans []types.ModelFilePlan, srcDigest, srcMediaType string) (digest.Digest, int64) {
	layers, err := pushLayerPlans(ctx, mag, rc, imgRef, plans)
	if err != nil {
		logrus.Fatal(err)
	}
	var subjSize int64
	if srcMd, found, e := mag.Database.ManifestGetByDigest(ctx, srcDigest); e == nil && found {
		subjSize = srcMd.Size
	}
	cfg, cfgDig, mani, maniDig, err := types.AssembleModelImage("", "", layers, srcDigest, srcMediaType, subjSize, nil)
	if err != nil {
		logrus.Fatal(err)
	}
	if err := pushAssembledImage(ctx, rc, imgRef, cfg, cfgDig, mani, maniDig); err != nil {
		logrus.Fatal(err)
	}
	logrus.Infof("pushed model-image models/%s-layer:%s (manifest %s)", rpName, rpQuantTag, maniDig)
	return maniDig, int64(len(mani))
}

// wireImageUpstream wires Upstream+Manifest DIRECTLY from the push-registered
// manifest so the manipulation engine resolves the pushed image (no re-fetch;
// metahub already registered the Manifest node on push).
func wireImageUpstream(ctx context.Context, mag magazine.Magazine, imageRefStr string, maniDig digest.Digest) error {
	up, err := types.NewUpstreamByName(imageRefStr)
	if err != nil {
		return err
	}
	up.Digest = maniDig // UpstreamGet matches by reg+repo+tag; digest only narrows.
	if _, err := mag.Database.UpstreamGetInsert(ctx, &up); err != nil {
		return fmt.Errorf("UpstreamGetInsert(%s): %w", imageRefStr, err)
	}
	md, found, err := mag.Database.ManifestGetByDigest(ctx, maniDig.String())
	if err != nil {
		return fmt.Errorf("ManifestGetByDigest(%s): %w", maniDig, err)
	}
	if !found {
		return fmt.Errorf("pushed manifest %s not found in graph (metahub push did not register it)", maniDig)
	}
	if err := mag.Database.ManifestLinkToUpstream(ctx, md.ID, up.ID); err != nil {
		return fmt.Errorf("ManifestLinkToUpstream(mani=%d, up=%d): %w", md.ID, up.ID, err)
	}
	return nil
}

// linkModelToProfile registers the quant as a variant of the model/<name>
// manipulation and attaches it to the pushed-under profile; returns the profile.
func linkModelToProfile(ctx context.Context, mag magazine.Magazine, maniDig digest.Digest) types.Profile {
	mpID, err := mag.Database.ModelQuantVariantUpsert(ctx, rpName, rpQuantTag, maniDig.String())
	if err != nil {
		logrus.Fatal(err)
	}
	prf, found, err := mag.Database.ProfileGetByLabel(ctx, rpProfile)
	if err != nil {
		logrus.Fatalf("ProfileGetByLabel(%s): %v", rpProfile, err)
	}
	if !found {
		logrus.Fatalf("profile %q not found — seed the profile before repackaging", rpProfile)
	}
	if err := mag.Database.ProfileLinkToManipulation(ctx, prf.ID, mpID); err != nil {
		logrus.Fatalf("ProfileLinkToManipulation(prf=%d, mp=%d): %v", prf.ID, mpID, err)
	}
	logrus.Infof("manipulation %s (id %d) linked to profile %q", yamlCfg.ModelManipulationLabel(rpName), mpID, rpProfile)
	return prf
}
