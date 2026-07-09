/*
Copyright © 2026 Christian Kniep <christian@qnib.org>
*/
package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/metahub-registry/metahub-admin/pkg/config"
	"github.com/metahub-registry/metahub-core/pkg/backends/magazine"
	"github.com/metahub-registry/metahub-core/pkg/types"
)

var (
	rtName     string
	rtImage    string
	rtFormats  string
	rtContract string
)

// runtimeCmd groups runtime-registry subcommands.
var runtimeCmd = &cobra.Command{
	Use:   "runtime",
	Short: "Manage runtime (application container) registrations",
}

// runtimeRegisterCmd registers/updates a runtime: the application container image
// a composition uses, plus its coarse compatibility (model formats it can run +
// the config contract it honors). Idempotent on --name.
var runtimeRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register (or update) a runtime and its model-format / config-contract compatibility",
	Run:   runtimeRegisterRun,
}

func init() {
	rootCmd.AddCommand(runtimeCmd)
	runtimeCmd.AddCommand(runtimeRegisterCmd)
	f := runtimeRegisterCmd.Flags()
	f.StringVar(&rtName, "name", "", "runtime name (the name a config's runtime annotation references)")
	f.StringVar(&rtImage, "image", "", "application container image ref")
	f.StringVar(&rtFormats, "formats", "", "comma-separated model formats the runtime can run (e.g. gguf)")
	f.StringVar(&rtContract, "config-contract", "", "config contract the runtime honors (empty = any)")
}

func runtimeRegisterRun(cmd *cobra.Command, args []string) {
	config.SetCommandValues(cmd)
	mag, ctx, err := magazine.NewMagazine(cmd, "")
	if err != nil {
		logrus.Fatal(err)
	}
	if rtName == "" {
		logrus.Fatal("--name is required")
	}
	rt := types.Runtime{
		Name:           rtName,
		Image:          rtImage,
		Formats:        types.FormatsFromCSV(rtFormats),
		ConfigContract: rtContract,
	}
	id, err := mag.Database.RuntimeUpsert(ctx, rt)
	if err != nil {
		logrus.Fatalf("RuntimeUpsert(%s): %v", rtName, err)
	}
	logrus.Infof("registered runtime %q (id %d): image=%s formats=%v config-contract=%q",
		rt.Name, id, rt.Image, rt.Formats, rt.ConfigContract)
}
