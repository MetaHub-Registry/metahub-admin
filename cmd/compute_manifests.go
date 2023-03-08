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

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// parseCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func cManifestsCmd(cmd *cobra.Command, args []string) {
	config.SetCommandValues(cmd)
	mag, err := magazine.NewMagazine(cmd, MH_TABLES)
	if err != nil {
		logrus.Error(err)
	}
	err = mag.ComputeManifests()
	if err != nil {
		logrus.Error(err)
	}

}
