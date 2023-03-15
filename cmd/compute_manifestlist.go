package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/metahub-registry/metahub-admin/pkg/config"
	"github.com/metahub-registry/metahub-core/pkg/backends/magazine"
)

var computeManifestlistsCmd = &cobra.Command{
	Use:   "manifestlists",
	Short: "Compute manifestlist for each Collection/Filter tupel",
	Run:   cManifestlistCmd,
}

func init() {
	computeCmd.AddCommand(computeManifestlistsCmd)
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// parseCmd.PersistentFlags().String("foo", "", "A help for foo")
	computeManifestlistsCmd.Flags().StringP("repo", "r", ".*", "Collection repository (regex) to compute manifestlist for")
	computeManifestlistsCmd.Flags().StringP("tag", "t", ".*", "Collection tag (regex) to compute manifestlist for")
	computeManifestlistsCmd.Flags().StringP("profile", "p", ".*", "Profile label (regex) to compute manifestlist for")
	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// parseCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func cManifestlistCmd(cmd *cobra.Command, args []string) {
	config.SetCommandValues(cmd)
	mag, ctx, err := magazine.NewMagazine(cmd, MH_TABLES)
	if err != nil {
		logrus.Error(err)
	}
	err = mag.ComputeManifestlists(ctx)
	if err != nil {
		logrus.Error(err)
	}

}
