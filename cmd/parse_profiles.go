package cmd

import (
	"fmt"

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
	config.SetCommandValues(cmd)
	if len(args) == 0 {
		logrus.Error("Please provide the directory to recursivly look for YAML files as the first argument")
	}
	prfs := make([]yamlCfg.Profile, 0)
	for _, a := range args {
		ps, err := yamlCfg.FindProfiles(a)
		if err != nil {
			logrus.Error(err)
		}
		prfs = append(prfs, ps...)
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
		p, _ := mag.Database.ProfileFromYAML(ctx, prf)
		err := mag.Database.ProfileGetInsert(ctx, &p)
		if err != nil {
			logrus.Fatal(err)
		}
	}

}
