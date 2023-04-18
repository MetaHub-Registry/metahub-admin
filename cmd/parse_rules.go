package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/metahub-registry/metahub-admin/pkg/config"
	"github.com/metahub-registry/metahub-core/pkg/backends/magazine"
	yamlCfg "github.com/metahub-registry/metahub-core/pkg/yaml/config"
)

var parseRulesCmd = &cobra.Command{
	Use:   "rules",
	Short: "Go through rules and put them into the graph database",
	Run:   pRulesCmd,
}

func init() {
	parseCmd.AddCommand(parseRulesCmd)
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// parseCmd.PersistentFlags().String("foo", "", "A help for foo")
	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// parseCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func pRulesCmd(cmd *cobra.Command, args []string) {
	config.SetCommandValues(cmd)
	if len(args) == 0 {
		logrus.Fatal("Please provide directories to recursivly look for YAML files as the first argument")

	}
	rs := make([]yamlCfg.Rule, 0)
	for _, a := range args {
		rls, err := yamlCfg.FindRules(a)
		if err != nil {
			logrus.Error(err)
		}
		rs = append(rs, rls...)
	}

	mag, ctx, err := magazine.NewMagazine(cmd, "")
	if err != nil {
		logrus.Error(err)
	}
	switch emptyGrah {
	case "all":
		logrus.Info("Empty complete graph")
		err = mag.Database.EmptyData(true)
		if err != nil {
			logrus.Error(err)
		}
	case "keep-upstream":
		logrus.Info("Empty graph, but keep upstream")
		err = mag.Database.EmptyData_KeepUpstream()
		if err != nil {
			logrus.Error(err)
		}
	case "none":
		logrus.Info("Do not empty graph")
	default:
		logrus.Fatalf("unkown emptyGraph option '%s'", emptyGrah)
	}
	if emptyMinio {
		logrus.Info("Empty minio")
		err = mag.Minio.EmptyAll()
		if err != nil {
			logrus.Error(err)
		}
	}
	for _, rl := range rs {
		r := rl.ToTypesRule()
		logrus.Infof("Insert Rule: %s:%s (%s)", r.Name, r.Tag, r.Type)
		_, err := mag.Database.Rule_GetCreate(ctx, r)
		if err != nil {
			logrus.Fatal(err)
		}
	}

}
