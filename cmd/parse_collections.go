package cmd

import (
	"context"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/metahub-registry/metahub-admin/pkg/backends/magazine"
	"github.com/metahub-registry/metahub-admin/pkg/config"
	yamlCfg "github.com/metahub-registry/metahub-admin/pkg/yaml/config"
)

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
	switch len(args) {
	case 1:
		dpath := args[0]
		if _, err := os.Stat(dpath); os.IsNotExist(err) {
			l.Errorf("Directory does not exists: %s", dpath)
		}
	default:
		l.Error("Please provide the directory to recursivly look for YAML files as the first argument")
	}
	cols, err := yamlCfg.FindManifestCollections(ctx, args[0])
	if err != nil {
		l.Error(err)
	}

	mag, err := magazine.NewMagazine(cmd)
	if err != nil {
		l.Error(err)
	}
	_ = mag
	for _, col := range cols {
		l.Infof("# Col: %s:%s", col.Name, col.Tag)
		// Create Collection
		colID, err := mag.CreateCollection(col)
		if err != nil {
			l.Error(err)
			continue
		}
		l.Infof("Collection: %d", colID)
		for _, m := range col.Manifests {
			l.Infof("  Maninfest > %s", m.Image)
			// Fetch real Manifest from cache
			mani, err := mag.GetManifest(ctx, m)
			mani.Platform.Features = strings.Join(m.Platform.Features, ",")
			if err != nil {
				l.Error(err)
				continue
			}
			//// 1. Manifest
			// 1. Insert Manifest
			maniID, err := mag.NeoInsertManifest(ctx, mani)
			mani.NeoID = maniID
			if err != nil {
				l.Error(err)
				continue
			}
			//l.Warnf("  ManifestID: %d", maniID)
			//// Link Collection to Manifest
			err = mag.NeoLinkCollectionToManifest(ctx, colID, maniID)
			if err != nil {
				l.Error(err)
				continue
			}
			// 2. Insert Upstream
			upID, err := mag.NeoInsertUpstream(ctx, mani.Upstream)
			if err != nil {
				l.Error(err)
				continue
			}
			//l.Warnf("  UpstreamID: %d", upID)
			//// Link Manifest to Upstream
			err = mag.NeoLinkManifestToUpstream(ctx, maniID, upID)
			if err != nil {
				l.Error(err)
				continue
			}
			// 3. Insert Config
			cfgID, err := mag.NeoInsertConfig(ctx, mani.Config, mani.Upstream)
			if err != nil {
				l.Error(err)
				continue
			}
			//l.Warnf("  configID: %d", cfgID)
			//// Link Manifest to Config
			err = mag.NeoLinkManifestToConfig(ctx, maniID, cfgID)
			if err != nil {
				l.Error(err)
				continue
			}
			// 4. Insert Layers
			for idx, layer := range mani.Layers {
				lID, err := mag.NeoInsertLayer(ctx, layer)
				if err != nil {
					l.Error(err)
					continue
				}
				//l.Warnf("  LayerID: %d", lID)
				//// Link Manifest to Layer
				err = mag.NeoLinkManifestToLayer(ctx, maniID, lID, idx)
				if err != nil {
					l.Error(err)
					continue
				}
			}
			// 5. Insert Platform
			pID, err := mag.NeoInsertPlatform(ctx, mani.Platform)
			if err != nil {
				l.Error(err)
				continue
			}
			//l.Warnf("  PlatformID (Features: %s): %d", mani.Platform.Features, pID)
			//// Link Manifest to Platform
			err = mag.NeoLinkManifestToPlatform(ctx, maniID, pID)
			if err != nil {
				l.Error(err)
				continue
			}
			//// Connect new Platform to Filter
			err = mag.NeoLinkPlatformToFilter(ctx, pID, mani.Platform)
			if err != nil {
				l.Error(err)
				continue
			}
		}
	}

}
