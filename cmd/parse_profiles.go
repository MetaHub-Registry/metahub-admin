package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/metahub-registry/metahub-admin/pkg/backends/magazine"
	"github.com/metahub-registry/metahub-admin/pkg/config"
	yamlCfg "github.com/metahub-registry/metahub-admin/pkg/yaml/config"
	"github.com/metahub-registry/metahub-core/pkg/types"
)

var parseProfilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "Go through profiles and put them into the graph database",
	Run:   pParseCmd,
}

func init() {
	parseCmd.AddCommand(parseProfilesCmd)
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// parseCmd.PersistentFlags().String("foo", "", "A help for foo")
	parseProfilesCmd.Flags().String("metahub-db", "", "The metahub database file to use - will be overwritten")
	parseProfilesCmd.Flags().Bool("empty-graph", false, "Remove all nodes and edges from the graph database")
	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// parseCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func pParseCmd(cmd *cobra.Command, args []string) {
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
	prfs, err := yamlCfg.FindProfiles(ctx, args[0])
	if err != nil {
		logrus.Error(err)
	}

	mag, err := magazine.NewMagazine(cmd, "")
	if err != nil {
		logrus.Error(err)
	}
	if mag.SQlite.MhFile == "" {
		emtpyGraph, _ := cmd.Flags().GetBool("empty-graph")
		if emtpyGraph {
			err = mag.Neo.EmptyGraph()
			if err != nil {
				logrus.Fatal(err)
			}
		}
	} else {
		defer mag.SQlite.Close()
	}
	for _, prf := range prfs {
		// Insert Filter
		f := types.Filter{
			Label: prf.Name,
		}
		feats := []string{}
		for _, act := range prf.Actions {
			for k, v := range act.Args {
				if k == "value" {
					feats = append(feats, v.(string))
				}
			}
		}
		f.Feature = strings.Join(feats, ",")
		msg := fmt.Sprintf("# Profile: %s", f.Label)
		if len(f.Feature) > 0 {
			msg += fmt.Sprintf(" [%s]", f.Feature)
		}
		logrus.Info(msg)
		// Filter
		var fID int64
		if mag.SQlite.MhFile == "" {
			///// NEO
			fID, err = mag.NeoInsertFilter(ctx, f)
			if err != nil {
				logrus.Error(err)
				continue
			}
		} else {
			fID, err = mag.SQliteInsertFilter(ctx, f)
			if err != nil {
				logrus.Error(err)
				continue
			}
		}
		p := types.Profile{
			Name: prf.Name,
		}
		var prfId int64
		if mag.SQlite.MhFile == "" {
			prfId, err = mag.NeoInsertProfile(ctx, p)
			if err != nil {
				logrus.Error(err)
				continue
			}
		} else {
			prfId, err = mag.SQliteInsertProfile(ctx, p)
			if err != nil {
				logrus.Error(err)
				continue
			}
		}
		//logrus.Warnf("Profile '%s' has ID %d", prf.Name, prfId)

		//logrus.Warnf("Filter '%s' has ID %d", f.Label, fId)
		// Link Profile and Filter
		if mag.SQlite.MhFile == "" {
			err = mag.NeoLinkProfileToFilter(ctx, prfId, fID)
			if err != nil {
				logrus.Error(err)
				continue
			}
		} else {
			err = mag.SQliteLinkProfileToFilter(ctx, prfId, fID)
			if err != nil {
				logrus.Error(err)
				continue
			}
		}
	}

}
