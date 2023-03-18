package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/metahub-registry/metahub-admin/pkg/config"
	"github.com/metahub-registry/metahub-core/pkg/backends/magazine"
)

var computeManifestsCmd = &cobra.Command{
	Use:   "manifests",
	Short: "Compute new config blobs and manifests if changes to the configurations are made (e.g. annotations)",
	Run:   cManifestsCmd,
}

func init() {
	computeCmd.AddCommand(computeManifestsCmd)
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// parseCmd.PersistentFlags().String("foo", "", "A help for foo")
	computeManifestsCmd.Flags().String("manifest", "", "manifest upstream to compute manifests for (e.g. dev:80/qnib/featuretest:latest, defaults to all)")
	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// parseCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func cManifestsCmd(cmd *cobra.Command, args []string) {
	config.SetCommandValues(cmd)
	mag, ctx, err := magazine.NewMagazine(cmd, MH_TABLES)
	if err != nil {
		logrus.Error(err)
	}
	ctx, span := tracer.Start(ctx, "cManifestsCmd")
	defer span.End()
	if m, ok := mag.Config.Args["manifest"]; ok && m != "" {
		mani, err := mag.Database.Manifest_GetByLabel(ctx, m)
		if err != nil {
			logrus.Fatal(err)
		}
		err = mag.ComputeManifest(ctx, mani)
		if err != nil {
			logrus.Fatal(err)
		}
		return
	}
	err = mag.ComputeManifests(ctx)
	if err != nil {
		logrus.Error(err)
	}

}
