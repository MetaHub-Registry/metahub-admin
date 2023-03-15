package cmd

import (
	"context"
	"os"

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
	rls, err := yamlCfg.FindRules(args[0])
	if err != nil {
		logrus.Error(err)
	}

	mag, ctx, err := magazine.NewMagazine(cmd, "")
	if err != nil {
		logrus.Error(err)
	}
	if emptyGrah {
		logrus.Info("Empty graph")
		err = mag.Database.EmptyData(false)
		if err != nil {
			logrus.Error(err)
		}
	}
	if emptyMinio {
		logrus.Info("Empty minio")
		err = mag.Minio.EmptyAll()
		if err != nil {
			logrus.Error(err)
		}
	}
	for _, rl := range rls {
		r := rl.ToTypesRule()
		logrus.Infof("Insert Rule: %s:%s (%s)", r.Name, r.Tag, r.Type)
		_, err := mag.Database.Rule_GetCreate(ctx, r)
		if err != nil {
			logrus.Fatal(err)
		}
	}

}
