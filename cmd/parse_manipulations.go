package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/metahub-registry/metahub-admin/pkg/config"
	"github.com/metahub-registry/metahub-core/pkg/backends/magazine"
	"github.com/metahub-registry/metahub-core/pkg/types"
	yamlCfg "github.com/metahub-registry/metahub-core/pkg/yaml/config"
)

var parseManipulationsCmd = &cobra.Command{
	Use:   "manipulations",
	Short: "Parses manipulations",
	Run:   pManipulationsCmd,
}

func init() {
	parseCmd.AddCommand(parseManipulationsCmd)
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// parseCmd.PersistentFlags().String("foo", "", "A help for foo")
	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// parseCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func pManipulationsCmd(cmd *cobra.Command, args []string) {
	config.SetCommandValues(cmd)
	if len(args) == 0 {
		logrus.Fatal("Please provide directories to recursivly look for YAML files as the first argument")

	}
	res := make([]yamlCfg.Manipulation, 0)
	for _, a := range args {
		mps, err := yamlCfg.FindManipulations(a)
		if err != nil {
			logrus.Error(err)
		}
		res = append(res, mps...)
	}

	mag, ctx, err := magazine.NewMagazine(cmd, "")
	if err != nil {
		logrus.Error(err)
	}

	if emptyMinio {
		logrus.Info("Empty minio")
		err = mag.Minio.EmptyAll()
		if err != nil {
			logrus.Error(err)
		}
	}

	for _, cfgMp := range res {
		/// Fetch Upstream first
		for _, mpl := range cfgMp.ManipuLayers {
			up, err := types.NewUpstreamByName(mpl.Image)
			if err != nil {
				logrus.Fatal(err)
			}
			if err = mag.CacheUpstream(ctx, up); err != nil {
				logrus.Fatal(err)
			}
		}
		mp, e := mag.Database.ManipulationFromYAML(ctx, cfgMp)
		if e != nil {
			logrus.Fatal(e)
		}
		logrus.Infof("Insert Manipulation: %s", mp.Label)
		mpID, err := mag.ManipulationGetCreate(ctx, mp)
		if err != nil {
			logrus.Fatal(err)
		}
		logrus.Infof("Manipulation %s has ID %d", mp.Label, mpID)
		// TODO: Check for upstream layers in Minio?

	}

}
