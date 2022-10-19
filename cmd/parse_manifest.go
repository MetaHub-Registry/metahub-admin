package cmd

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/metahub-registry/metahub-admin/pkg/backends/magazine"
	"github.com/metahub-registry/metahub-admin/pkg/config"
	yamlCfg "github.com/metahub-registry/metahub-admin/pkg/yaml/config"
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
	parseManifestCmd.Flags().Bool("remove-manifests-alias", false, "Remove alias object in manifest (<reg>/<repo>/<tag> -> sha256)")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// parseCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func pManiCmd(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	config.SetCommandValues(cmd)
	switch len(args) {
	case 1:
		dpath := args[0]
		if _, err := os.Stat(dpath); os.IsNotExist(err) {
			logrus.Errorf("Directory does not exists: %s", dpath)
		}
	default:
		logrus.Error("Please provide the directory to recursivly look for YAML files as the first argument")
	}
	cols, err := yamlCfg.FindManifestCollections(ctx, args[0])
	if err != nil {
		logrus.Error(err)
	}

	mag, err := magazine.NewMagazine(cmd, MH_TABLES)
	if err != nil {
		logrus.Error(err)
	}
	emtpyManiAlias, _ := cmd.Flags().GetBool("remove-manifests-alias")
	if emtpyManiAlias {
		err = mag.Minio.RemoveManifestAliases()
		if err != nil {
			logrus.Fatal(err)
		}
	}
	for _, col := range cols {
		logrus.Infof("# Col: %s:%s", col.Name, col.Tag)
		for _, m := range col.Manifests {
			logrus.Infof("  > %s", m.Image)
			// Make sure manifest is cached (manifest, config, layers)
			mani, err := mag.CachManifest(ctx, m)
			if err != nil {
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
			// Deal with Layers
			err = mag.CacheLayers(ctx, mani)
			if err != nil {
				logrus.Error(err)
				continue

			}

		}
	}

}
