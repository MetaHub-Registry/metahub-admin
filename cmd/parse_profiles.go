package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/metahub-registry/metahub-admin/pkg/config"
	"github.com/metahub-registry/metahub-core/pkg/backends/magazine"
	yamlCfg "github.com/metahub-registry/metahub-core/pkg/yaml/config"
)

var parseProfilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "Go through profiles and put them into the graph database",
	Run:   pProfilesCmd,
}

func init() {
	parseCmd.AddCommand(parseProfilesCmd)
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// parseCmd.PersistentFlags().String("foo", "", "A help for foo")
	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// parseCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func pProfilesCmd(cmd *cobra.Command, args []string) {
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
	prfs, err := yamlCfg.FindProfiles(args[0])
	if err != nil {
		logrus.Error(err)
	}

	mag, ctx, err := magazine.NewMagazine(cmd, "")
	if err != nil {
		logrus.Error(err)
	}
	for _, prf := range prfs {
		label := fmt.Sprintf("%s:%s", prf.Name, prf.Tag)
		if prf.Name == "" {
			label = prf.Tag
		}
		logrus.Infof("Insert profile: %s", label)
		p, _ := prf.ToProfile()
		err := mag.Database.Profile_GetInsert(ctx, &p)
		if err != nil {
			logrus.Fatal(err)
		}
	}

}
