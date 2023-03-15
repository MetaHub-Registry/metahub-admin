package cmd

import (
	"context"
	"os"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/metahub-registry/metahub-admin/pkg/config"
	"github.com/metahub-registry/metahub-core/pkg/backends/magazine"
	"github.com/metahub-registry/metahub-core/pkg/types"
	yamlCfg "github.com/metahub-registry/metahub-core/pkg/yaml/config"
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

	mag, ctx, err := magazine.NewMagazine(cmd, MH_TABLES)
	if err != nil {
		l.Error(err)
	}
	_ = mag
	for _, col := range cols {
		l.Infof("# Col: %s:%s", col.Name, col.Tag)
		// Create Collection
		colID, colTagID, err := mag.CreateCollection(col)
		if err != nil {
			l.Error(err)
			continue
		}
		l.Infof("Collection: %d / Tag: %d", colID, colTagID)
		for _, m := range col.Manifests {
			l.Infof("  Maninfest > %s", m.Image)
			// Fetch real Manifest from cache
			mani, err := mag.GetManifest(ctx, m)
			if err != nil {
				err = errors.Wrap(err, "Failed to get Manifest from cache using GetManifest()")
				l.Error(err)
				continue
			}
			// Make sure the (standin) Manifest is present
			ms := types.ManifestFromManifestData(mani)
			msID, err := mag.Database.Manifest_GetInsert(ctx, ms)
			if err != nil {
				l.Error(err)
				continue
			}
			// 1. Insert Platform
			platID, err := mag.NeoInsertPlatform(ctx, mani.Platform)
			if err != nil {
				l.Error(err)
				continue
			}
			err = mag.Database.LinkManifestToPlatform(ctx, msID, platID)
			if err != nil {
				l.Error(err)
				continue
			}
			/// 1. Insert Config
			cfgID, err := mag.InsertConfig(ctx, mani.Config, mani.Upstream)
			if err != nil {
				l.Error(err)
				continue
			}
			l.Warnf("  configID: %d", cfgID)
			// 4. Insert Layers
			for _, layer := range mani.Layers {
				lID, err := mag.NeoInsertLayer(ctx, layer)
				if err != nil {
					l.Error(err)
					continue
				}
				l.Warnf("  LayerID: %d", lID)
				//// Link Manifest to Layer
				//err = mag.NeoLinkManifestToLayer(ctx, maniID, lID, idx)
				//if err != nil {
				//	l.Error(err)
				//	continue
				//}
			}
			//// 1. Manifest
			// 1. Insert Manifest
			maniID, err := mag.NeoInsertManifestData(ctx, msID, mani, "original")
			mani.ID = maniID
			if err != nil {
				l.Error(err)
				continue
			}
			//l.Warnf("  ManifestID: %d", maniID)
			//// Link Collection to Manifest
			err = mag.NeoLinkCollectionToManifest(ctx, colID, colTagID, msID)
			if err != nil {
				l.Error(err)
				continue
			}
			//// Link Manifest to Config
			err = mag.NeoLinkManifestDataToConfig(ctx, maniID, cfgID)
			if err != nil {
				l.Error(err)
				continue
			}
			/// Container Image Labels (from Config Blob)
			for k, v := range mani.Config.Cfg.Config.Labels {
				logrus.Warnf("  Label: %s=%s", k, v)
				a := types.NewAnnotation(k, v)
				mag.Config_AddLabel(ctx, cfgID, a)
			}

			///// Annotations
			// -> Extract the extra Annotations from the Collection per Manifest
			// -> Add them to the ManifestStandin (msID
			for k, v := range m.Annotations {
				a := types.NewAnnotation(k, v)
				err = mag.Database.Manifest_AddAnnotation(ctx, msID, &a)
				if err != nil {
					// Fail if Annotation is already defined and has a different value
					l.Error(err)
					return
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
				mag.Database.Manifest_AddAnnotations(ctx, maniID, &mani.Annotations)
			*/

		}
	}

}
