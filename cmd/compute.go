/*
Copyright © 2022 Christian Kniep <christian@qnib.org>
*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.com/metahub-registry/metahub-admin/pkg/config"
)

// computeCmd represents the compute command
var computeCmd = &cobra.Command{
	Use:   "compute",
	Short: "Parse YAML files (profiles, collections, base, manipulation) and store them in the database",
	Run:   cCmd,
}

func init() {
	rootCmd.AddCommand(computeCmd)

}

func cCmd(cmd *cobra.Command, args []string) {
	config.SetCommandValues(cmd)
}
