package cmd

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/metahub-registry/metahub-admin/pkg/config"
	"github.com/metahub-registry/metahub-core/pkg/backends/magazine"
	yamlCfg "github.com/metahub-registry/metahub-core/pkg/yaml/config"
)

var parseManifestCmd = &cobra.Command{
	Use:   "manifests",
	Short: "Go through collections, parse manifests and make sure the blobs are cached",
	Run:   pManiCmd,
}

func init() {
	parseCmd.AddCommand(parseManifestCmd)
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// parseCmd.PersistentFlags().String("foo", "", "A help for foo")
	parseManifestCmd.Flags().Bool("skip-layer-download", false, "Do not download layers to object store")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// parseCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func pManiCmd(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	config.SetCommandValues(cmd)
	if len(args) == 0 {
		logrus.Error("Please provide the directory to recursivly look for YAML files as the first argument")
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
		logrus.Error(err)
	}
	skipLayerDownload, err := cmd.Flags().GetBool("skip-layer-download")
	if err != nil {
		skipLayerDownload = false
	}
	for _, col := range cols {
		logrus.Infof("# Col: %s:%s", col.Name, col.Tag)
		for _, m := range col.Manifests {
			logrus.Infof("  > %s", m.Image)
			// Make sure manifest is cached (manifest, config, layers)
			mani, err := mag.CachManifest(ctx, m)
			if err != nil {
				if err.Error() == "manifest type is not supported: application/vnd.docker.distribution.manifest.v1+prettyjws" {
					logrus.Warnf(" SKIP > %s", err)
					continue
				}
				logrus.Error(err)
				continue
			}
			logrus.Infof("	- %s", mani.Config.Digest)
			// Deal with ImageConfig
			err = mag.CachImageConfig(ctx, &mani)
			if err != nil {
				logrus.Error(err)
				continue
			}
			if !skipLayerDownload {
				// Deal with Layers
				err = mag.CacheLayers(ctx, mani)
				if err != nil {
					logrus.Error(err)
					continue

				}
			} else {
				logrus.Infof("  - Skip layer download")
			}
		}
	}

}
