package cmd

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/metahub-registry/metahub-admin/pkg/config"
	"github.com/metahub-registry/metahub-core/pkg/backends/magazine"
	"github.com/metahub-registry/metahub-core/pkg/types"
	"github.com/metahub-registry/metahub-core/pkg/utils"
	yamlCfg "github.com/metahub-registry/metahub-core/pkg/yaml/config"
)

// genericProfileLabel is the default profile that parsed manifests are
// attributed to. Mirrors metahub-api's defaultProfile constant.
const genericProfileLabel = "generic"

var parseCollectionCmd = &cobra.Command{
	Use:   "collections",
	Short: "Go through collections and push them into the Graph; caching must be done before",
	Run:   pCollectonCmd,
}

func init() {
	parseCmd.AddCommand(parseCollectionCmd)
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// parseCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// parseCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func pCollectonCmd(cmd *cobra.Command, args []string) {
	l := logrus.WithFields(logrus.Fields{
		"package": "cmd/parse_collections",
		"func":    "pCollectonCmd",
	})
	ctx := context.Background()
	config.SetCommandValues(cmd)
	if len(args) == 0 {
		l.Error("Please provide the directory to recursivly look for YAML files as the first argument")
	}
	cols := make(map[string]yamlCfg.ManifestCollection, 0)
	for _, a := range args {
		cs, err := yamlCfg.FindManifestCollections(ctx, a)
		if err != nil {
			logrus.Error(err)
		}
		for _, c := range cs {
			cols[c.Name] = c
		}
	}

	mag, ctx, err := magazine.NewMagazine(cmd, MH_TABLES)
	if err != nil {
		l.Error(err)
	}
	_ = mag

	// Resolve the generic profile once. If it is not seeded, Slot wiring
	// will be skipped per-manifest and parse falls back to writing only
	// the legacy wrapper edges (see fix-parse-collections-slot-wiring D4).
	// This keeps parse usable on fresh DBs where `parse profiles` has not
	// yet been run.
	genericPrf, genericPrfFound, prfErr := mag.Database.ProfileGetByLabel(ctx, genericProfileLabel)
	if prfErr != nil {
		l.Warnf("failed to resolve '%s' profile: %v; Slot wiring will be skipped", genericProfileLabel, prfErr)
		genericPrfFound = false
	}
	if !genericPrfFound {
		l.Warnf("profile '%s' not seeded; Slot wiring will be skipped. Run `metahub-admin parse profiles` first for pullable output", genericProfileLabel)
	}

	for _, col := range cols {
		l.Infof("# Col: %s:%s", col.Name, col.Tag)
		// Create Collection
		colID, colTagID, err := mag.CreateCollection(ctx, col)
		if err != nil {
			l.Error(err)
			continue
		}
		l.Infof("Collection: %d / Tag: %d", colID, colTagID)
		for _, m := range col.Manifests {
			l.Infof("  Manifest > [%s] %s", utils.ShortenDigest(m.Digest), m.Image)
			// Fetch real Manifest from cache
			mani, err := mag.GetManifest(ctx, m)
			if err != nil {
				err = errors.Wrap(err, "Failed to get Manifest from cache using GetManifest()")
				l.Error(err)
				continue
			}

			// Insert Config and Layers BEFORE ManifestGetInsert —
			// the new ManifestGetInsert validates that all referenced
			// blobs (config + layers) exist in the graph.
			cfgID, err := mag.InsertConfig(ctx, mani.Config, mani.Upstream)
			if err != nil {
				l.Error(err)
				continue
			}
			mani.Config.ID = cfgID
			l.Warnf("  configID: %d", cfgID)
			for _, layer := range mani.Layers {
				lID, err := mag.NeoInsertLayer(ctx, layer)
				if err != nil {
					l.Error(err)
					continue
				}
				l.Warnf("  LayerID: %d", lID)
			}

			// Make sure the Manifest node is present (get-or-create)
			if err = mag.Database.ManifestGetInsert(ctx, &mani); err != nil {
				l.Error(err)
				continue
			}
			//// Insert Manifest content node
			maniDataID, err := mag.NeoInsertManifest(ctx, mani.ID, mani, "parsed")
			mani.ID = maniDataID
			if err != nil {
				l.Error(err)
				continue
			}
			// Insert Platform
			platID, err := mag.NeoInsertPlatform(ctx, mani.Platform)
			if err != nil {
				l.Error(err)
				continue
			}
			err = mag.Database.LinkManifestToPlatform(ctx, maniDataID, platID)
			if err != nil {
				l.Error(err)
				continue
			}
			//// Link Collection to Manifest
			err = mag.NeoLinkCollectionToManifest(ctx, colID, colTagID, mani.ID)
			if err != nil {
				l.Error(err)
				continue
			}
			//// Link Manifest to Config
			err = mag.NeoLinkManifestToConfig(ctx, maniDataID, mani.Config)
			if err != nil {
				l.Error(err)
				continue
			}
			/// Container Image Labels (from Config Blob)
			for k, v := range mani.Config.Cfg.Config.Labels {
				logrus.Warnf("  Label: %s=%s", k, v)
				a := types.NewAnnotation(k, v)
				mag.ConfigAddLabel(ctx, cfgID, a)
			}

			///// Annotations
			as := types.NewAnnotations()
			for k, v := range m.Annotations {
				as.AddNewAnnotation(k, v)
			}
			if err = mag.Database.ManifestAddAnnotations(ctx, maniDataID, &as); err != nil {
				l.Error(err)
				continue
			}

			//// Slot wiring (fix-parse-collections-slot-wiring / task 2.2)
			// Mirror metahub-api's push-path wireSlotOnPush + profile-link
			// + rule-unwind sequence. These edges are what the current
			// Slot-centric ManifestExists lookup path walks, so without
			// them parsed content is unpullable via OCI GET/HEAD.
			//
			// Delegates entirely to existing metalib helpers — no new
			// primitives introduced, so any future changes in the push
			// path (e.g. manipulated-Manifest wiring) propagate to parse
			// automatically.
			if genericPrfFound {
				slot, slotErr := mag.Database.SlotGetCreate(ctx, col.Name, col.Tag, genericProfileLabel)
				if slotErr != nil {
					l.Errorf("SlotGetCreate(%s:%s#%s) failed: %v", col.Name, col.Tag, genericProfileLabel, slotErr)
				} else {
					if pushErr := mag.Database.SlotAppendPushedManifest(ctx, slot.ID, maniDataID); pushErr != nil {
						l.Errorf("SlotAppendPushedManifest(slot=%d, manifest=%d) failed: %v", slot.ID, maniDataID, pushErr)
					}
					if linkErr := mag.Database.LinkManifestToProfile(ctx, maniDataID, genericPrf.ID, "PULLED_BY"); linkErr != nil {
						l.Errorf("LinkManifestToProfile(manifest=%d, profile=%d, PULLED_BY) failed: %v", maniDataID, genericPrf.ID, linkErr)
					}
					if unwErr := mag.Database.ProfileUnwindRules(ctx, genericPrf.ID, maniDataID); unwErr != nil {
						l.Errorf("ProfileUnwindRules(profile=%d, manifest=%d) failed: %v", genericPrf.ID, maniDataID, unwErr)
					}
					l.Infof("wired Slot %d for %s:%s (manifest=%d, profile=%s)", slot.ID, col.Name, col.Tag, maniDataID, genericProfileLabel)
				}
			}

			//// Explicit collection profile (collection-profile-authority)
			// When the collection entry names a profile, bind this
			// manifest to it authoritatively so it is servable under that
			// profile (e.g. the zen2 image under `cpu#zen2`). An unknown
			// profile label is a hard error — a mis-authored collection
			// must surface loudly. Empty profile = no-op, falls back to
			// the generic/image-annotation path above.
			if m.Profile != "" {
				if bindErr := mag.BindManifestToProfile(ctx, col.Name, col.Tag, m.Profile, maniDataID); bindErr != nil {
					l.Errorf("BindManifestToProfile(%s:%s#%s, manifest=%d) failed: %v", col.Name, col.Tag, m.Profile, maniDataID, bindErr)
					continue
				}
			}

			/*****
			We are going to stay clean while parsing the collections:
			  1. Only add labels to the config object
			  2. Only add annotations to the manifest(standin) object
			during the `compute manifest` step, we'll add
			  1. Labels from the config object to the manifestData object if they do have the reverseDNS annotation
			  2. Annotations from the manifest(standin) object to the manifestData object.
			    -> if they conflict with the labels, we'll throw an error
				err = mag.Database.Config_AddLabels(ctx, cfgID, &mani.Annotations)
				if err != nil {
					l.Error(err)
					continue
				}
				mag.Database.ManifestAddAnnotations(ctx, maniID, &mani.Annotations)
			*/

		}

		//// Per-collection index compute (fix-parse-collections-slot-wiring / task 2.3)
		// Mirrors the push-path computeAfterManifestPush call. Rebuilds
		// the per-profile :Index from the Slot's HAS_PUSHED manifests,
		// so `GET /v2/<col.Name>/manifests/<col.Tag>` serves the computed
		// index right after parse completes (no separate `metahub-admin
		// compute manifestlists` step required).
		//
		// Non-fatal by design (D4): if rules are not seeded yet the
		// compute may legitimately produce no index, and the user can
		// rerun compute later without re-parsing.
		// Per-collection index compute. A single compute with the generic
		// pusher rebuilds the :Index for EVERY profile (generic + every
		// declared cpu#zenX) via rebuildIndexesPerProfile→ProfileGetAll,
		// using the scoped annotations BindManifestToProfile already
		// injected per declared manifest.
		//
		// IMPORTANT: do NOT call ComputeManifestlistsForRepoTag once per
		// declared profile. Its injectPusherAnnotations step broadcasts
		// the pusher profile's annotations onto EVERY manifest in the tag
		// (correct for `generic`, where all manifests are members). Calling
		// it for cpu#zen2 then cpu#zen3 cross-contaminates: every manifest
		// ends up with both microarch annotations, so every image matches
		// both zen profiles. The declared-profile binding is already done,
		// scoped, by BindManifestToProfile above; the indexes fall out of
		// this single generic compute.
		if genericPrfFound {
			if cmpErr := mag.ComputeManifestlistsForRepoTag(ctx, col.Name, col.Tag, &genericPrf); cmpErr != nil {
				l.Warnf("ComputeManifestlistsForRepoTag(%s:%s, profile=%s) failed (non-fatal): %v", col.Name, col.Tag, genericProfileLabel, cmpErr)
			}
		}
	}

}
