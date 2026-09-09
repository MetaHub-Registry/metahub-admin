package cmd

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/opencontainers/go-digest"
	"github.com/regclient/regclient/types/ref"

	"github.com/metahub-registry/metahub-core/pkg/backends/magazine"
	"github.com/metahub-registry/metahub-core/pkg/backends/minio"
	"github.com/metahub-registry/metahub-core/pkg/backends/neo"
	"github.com/metahub-registry/metahub-core/pkg/types"
	"github.com/metahub-registry/metahub-core/pkg/upstream"
)

// TestRepackageOnceModelOnlyIntegration is the command-level test of the
// `repackage` flow (implement-model-repackage §3.1): a fake single-file GGUF
// ORAS artifact is seeded into Neo4j + MinIO testcontainers, the model-layer
// image is pushed to a real OCI registry container, and the graph is asserted
// for the pushed manifest's upstream wiring plus the persisted `model/<name>`
// quant-variant manipulation linked to the push profile.
//
// Skipped under -short (test.sh runs `go test -short -timeout 60s`): needs
// Docker. Run explicitly with:
//
//	go test ./cmd -run TestRepackageOnceModelOnlyIntegration -v
//
// The one deliberate shim is the manifest registration between the registry
// push and the graph wiring: in production metahub's manifest-PUT handler
// registers the Manifest node when repackage pushes to it. The API is not part
// of this test's scope (the GoCD e2e covers the live path), so the test MERGEs
// a Manifest node by the pushed digest to stand in for that handler — exactly
// what wireImageUpstream's "metahub push did not register it" error guards on.
func TestRepackageOnceModelOnlyIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test: needs Docker (neo4j, minio, registry testcontainers)")
	}
	ctx := context.Background()

	/// Neo4j
	neoC, err := neo.SetupNeo4j(ctx, "")
	if err != nil {
		t.Fatalf("SetupNeo4j: %v", err)
	}
	defer neoC.Terminate(ctx)
	gdb := neo.NewNeo(neo.NeoConfig{
		DbName: "test",
		User:   "neo4j",
		Passwd: "test",
		URL:    neoC.BoltURI,
		HTTP:   neoC.HttpURI,
	})
	if err := gdb.Connect(); err != nil {
		t.Fatalf("neo connect: %v", err)
	}
	if err := gdb.Setup(ctx); err != nil {
		t.Fatalf("neo setup: %v", err)
	}

	/// MinIO
	minioC, err := minio.SetupMinio(ctx, "")
	if err != nil {
		t.Fatalf("SetupMinio: %v", err)
	}
	defer minioC.Terminate(ctx)
	blobKey := minio.GetBlobBucketKey()
	manifestsKey := minio.GetManifestsBucketKey()
	bc := map[string]minio.BucketConfig{
		blobKey:      minio.NewBucketConfig(blobKey, minioC.HttpURI, minioC.ConsoleURI),
		manifestsKey: minio.NewBucketConfig(manifestsKey, minioC.HttpURI, minioC.ConsoleURI),
	}
	mn, err := minio.NewMiniWithCfg(bc)
	if err != nil {
		t.Fatalf("NewMiniWithCfg: %v", err)
	}
	if err := mn.ConnectClients(); err != nil {
		t.Fatalf("minio connect: %v", err)
	}
	mn.BucketCheck()

	/// Registry (the push target)
	regC, err := upstream.SetupRegistry(ctx)
	if err != nil {
		t.Fatalf("SetupRegistry: %v", err)
	}
	defer regC.Terminate(ctx)

	mag := magazine.Magazine{Minio: mn, Database: gdb}

	// --- Seed the source GGUF blob in MinIO (content-addressed by digest) ---
	payload := append([]byte("GGUF"), bytes.Repeat([]byte("model-weights-"), 128)...)
	ggufDig := digest.FromBytes(payload).String()
	if err := mn.PutBlob(ctx, payload, ggufDig); err != nil {
		t.Fatalf("seed GGUF blob: %v", err)
	}

	// --- Seed the ORAS artifact in the graph (what a metahub ORAS push leaves
	//     behind): Repo <- Slot -HAS_PUSHED-> Manifest -USES-> config Blob and
	//     -HAS_LAYER-> the GGUF layer blob carrying the title annotation. ---
	const (
		rpArtifactRepoT = "models/qwen-test"
		rpArtifactTypeT = "application/vnd.qnib.model.artifact"
		rpNameT         = "qwen-test"
		rpQuantTagT     = "q8-kxl"
		rpProfileT      = "vram#12gb"
	)
	artDig := digest.FromString("fake-oras-artifact-manifest").String()
	cfgDig := digest.FromString("{}").String()
	seedQ := fmt.Sprintf(`
		CREATE (c:Repo {repo: %q})
		CREATE (md:Manifest {digest: %q, mediaType: "application/vnd.oci.image.manifest.v1+json", artifactType: %q, size: 512})
		CREATE (cfg:Blob {digest: %q, mediaType: %q, type: "config", size: 2})
		CREATE (md)-[:USES]->(cfg)
		CREATE (b:Blob {digest: %q, mediaType: "application/octet-stream", type: "layer", size: %d})
		CREATE (md)-[:HAS_LAYER {order: 0, mediaType: "application/octet-stream"}]->(b)
		CREATE (a:Annotation {key: "org.opencontainers.image.title", value: "qwen-q8.gguf"})
		CREATE (b)-[:HAS_ANNOTATION]->(a)
		WITH md, c
		MERGE (c)<-[:OF_REPO]-(s:Slot {repo: %q, tag: "latest", profile: "generic"})
		CREATE (s)-[:HAS_PUSHED]->(md)
		RETURN id(md) AS mdID`,
		rpArtifactRepoT, artDig, rpArtifactTypeT, cfgDig, types.ModelConfigMediaType, ggufDig, int64(len(payload)), rpArtifactRepoT)
	res, err := gdb.Execute(seedQ)
	if err != nil {
		t.Fatalf("seed ORAS artifact: %v", err)
	}
	if _, err := res.GetInt64("mdID"); err != nil {
		t.Fatalf("seed ORAS artifact: no manifest id returned: %v", err)
	}

	// --- Seed the profile the model is pushed under (rule property is part of
	//     the seed's Profile shape; ProfileGetByLabel returns it unconditionally) ---
	if _, err := gdb.Execute(fmt.Sprintf(
		`MERGE (p:Profile {label: %q, name: "vram", tag: "12gb", rule: ""})`, rpProfileT)); err != nil {
		t.Fatalf("seed profile: %v", err)
	}

	// --- Point the rp* globals at the containers ---
	orig := captureRpGlobals()
	defer restoreRpGlobals(orig)
	rpArtifactRepo, rpArtifactType = rpArtifactRepoT, rpArtifactTypeT
	rpName, rpQuantTag, rpProfile, rpRuntime = rpNameT, rpQuantTagT, rpProfileT, ""
	rpTargetPath, rpRegistryURL, rpRegistryUser, rpRegistryPass = "", regC.HttpURI, "", ""
	rpRegistryTLS, rpSkipManip, rpConfigFile = false, false, ""

	// --- Phase A: read the artifact + push the model-layer image (the real
	//     code under test). This is repackageOnce steps 1–4. ---
	art, err := mag.Database.OrasArtifactGetByName(ctx, rpArtifactRepo, rpArtifactType)
	if err != nil {
		t.Fatalf("OrasArtifactGetByName: %v", err)
	}
	if len(art.Blobs) != 1 {
		t.Fatalf("expected 1 artifact blob (the GGUF), got %d", len(art.Blobs))
	}
	if art.Blobs[0].Digest != ggufDig || art.Blobs[0].Title != "qwen-q8.gguf" {
		t.Fatalf("unexpected blob %+v", art.Blobs[0])
	}
	plans, err := types.PlanModelRepackage(art.Blobs, rpName, rpTargetPath)
	if err != nil {
		t.Fatalf("PlanModelRepackage: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("expected 1 layer plan, got %d", len(plans))
	}

	imgRefStr := fmt.Sprintf("%s/models/%s-layer:%s", rpRegistryURL, rpName, rpQuantTag)
	imgRef, err := ref.New(imgRefStr)
	if err != nil {
		t.Fatalf("ref.New(%s): %v", imgRefStr, err)
	}
	rc := newPushClient(rpRegistryURL, "", "", false)
	defer rc.Close(ctx, ref.Ref{})

	maniDig, _ := pushModelLayerImage(ctx, mag, rc, imgRef, plans, art.Digest, art.MediaType)

	// --- Shim: register the pushed manifest + its layer blob (metahub API
	//     behaviour, see above): the handler records the Manifest node, the
	//     layer Blob, and the HAS_LAYER edge so reads can walk the image. ---
	m, err := rc.ManifestGet(ctx, imgRef)
	if err != nil {
		t.Fatalf("registry does not serve the pushed manifest: %v", err)
	}
	pushedDig := m.GetDescriptor().Digest
	if pushedDig != maniDig {
		t.Fatalf("registry manifest %s != pushed manifest %s", pushedDig, maniDig)
	}
	layerDig, layerSize, err := hashModelLayer(ctx, mag, plans[0])
	if err != nil {
		t.Fatalf("recompute layer digest: %v", err)
	}
	shimQ := fmt.Sprintf(`
		MERGE (md:Manifest {digest: %q})
		  ON CREATE SET md.mediaType = "application/vnd.oci.image.manifest.v1+json", md.size = 512
		MERGE (b:Blob {digest: %q})
		  ON CREATE SET b.mediaType = %q, b.type = "layer", b.size = %d, b.label = "shim-layer"
		MERGE (md)-[r:HAS_LAYER]->(b) ON CREATE SET r.order = 0, r.mediaType = %q`,
		maniDig.String(), layerDig.String(), types.ModelLayerMediaType, layerSize, types.ModelLayerMediaType)
	if _, err := gdb.Execute(shimQ); err != nil {
		t.Fatalf("shim push registration: %v", err)
	}

	// --- Phase B: wire + attach (repackageOnce steps 5–7, the real code). ---
	if err := wireImageUpstream(ctx, mag, imgRefStr, maniDig); err != nil {
		t.Fatalf("wireImageUpstream: %v", err)
	}
	prf := linkModelToProfile(ctx, mag, maniDig)
	if prf.Label != rpProfileT {
		t.Fatalf("linkModelToProfile returned profile %q, want %q", prf.Label, rpProfileT)
	}

	// --- Assertions ---

	// 1. The registry serves the pushed image.
	if _, err := rc.ManifestHead(ctx, imgRef); err != nil {
		t.Fatalf("ManifestHead on pushed image: %v", err)
	}

	// 2. Graph: profile -USES-> model/<name> manipulation -HAS_VARIANT[quant]->
	//    ManipuConfig -HAS_LAYER-> ManipuLayer -USES-> the pushed Manifest,
	//    and the ManipuLayer points UPSTREAM at models/<name>-layer:<quant>.
	linkQ := fmt.Sprintf(`
		MATCH (p:Profile {label: %q})-[:USES]->(mp:Manipulation {label: "model/%s"})
		MATCH (mp)-[:HAS_VARIANT {rule: "quant:%s"}]->(mc:ManipuConfig)-[:HAS_LAYER]->(mpl:ManipuLayer {label: "model/%s"})
		MATCH (mpl)-[:UPSTREAM]->(up:Upstream {repo: "models/%s-layer", tag: %q})
		RETURN id(mp) AS mpID, id(mpl) AS mplID`,
		rpProfileT, rpNameT, rpQuantTagT, rpNameT, rpNameT, rpQuantTagT)
	rows, err := gdb.Query(linkQ)
	if err != nil {
		t.Fatalf("manipulation link query: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 profile→model-manipulation→variant→layer→upstream row, got %d", len(rows))
	}
	mpID, err := rows[0].GetInt64("mpID")
	if err != nil {
		t.Fatalf("read mpID: %v", err)
	}

	// 3. The model resolver sees the quant.
	quants, err := gdb.ModelQuantsGetByName(ctx, rpNameT)
	if err != nil {
		t.Fatalf("ModelQuantsGetByName: %v", err)
	}
	if len(quants) != 1 || quants[0].Tag != rpQuantTagT {
		t.Fatalf("ModelQuantsGetByName = %+v, want one %s quant", quants, rpQuantTagT)
	}
	if quants[0].Digest != maniDig.String() {
		t.Fatalf("quant digest %s != pushed manifest %s", quants[0].Digest, maniDig)
	}

	// 4. Idempotent re-run (2.8): re-running the attach against the same push
	//    must be a no-op — same manipulation id, still exactly one variant.
	prf2 := linkModelToProfile(ctx, mag, maniDig)
	rows2, err := gdb.Query(linkQ)
	if err != nil {
		t.Fatalf("manipulation link query (re-run): %v", err)
	}
	if len(rows2) != 1 {
		t.Fatalf("re-run must not duplicate the variant chain, got %d rows", len(rows2))
	}
	mpID2, _ := rows2[0].GetInt64("mpID")
	if mpID2 != mpID {
		t.Fatalf("re-run created a new manipulation: id %d → %d (want idempotent %d)", mpID2, mpID, mpID)
	}
	if prf2.ID != prf.ID {
		t.Fatalf("re-run resolved a different profile: %d vs %d", prf2.ID, prf.ID)
	}
}

// captureRpGlobals snapshots the rp* flag globals so the test can restore them.
func captureRpGlobals() func() {
	oArtifactRepo, oArtifactType, oName, oQuantTag := rpArtifactRepo, rpArtifactType, rpName, rpQuantTag
	oProfile, oRuntime, oTargetPath := rpProfile, rpRuntime, rpTargetPath
	oRegistryURL, oRegistryUser, oRegistryPass := rpRegistryURL, rpRegistryUser, rpRegistryPass
	oRegistryTLS, oSkipManip, oConfigFile := rpRegistryTLS, rpSkipManip, rpConfigFile
	return func() {
		rpArtifactRepo, rpArtifactType, rpName, rpQuantTag = oArtifactRepo, oArtifactType, oName, oQuantTag
		rpProfile, rpRuntime, rpTargetPath = oProfile, oRuntime, oTargetPath
		rpRegistryURL, rpRegistryUser, rpRegistryPass = oRegistryURL, oRegistryUser, oRegistryPass
		rpRegistryTLS, rpSkipManip, rpConfigFile = oRegistryTLS, oSkipManip, oConfigFile
	}
}

func restoreRpGlobals(restore func()) {
	restore()
}
