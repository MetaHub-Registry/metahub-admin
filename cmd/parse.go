/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/metahub-registry/metahub-admin/pkg/config"
)

// parseCmd represents the parse command
var (
	parseCmd = &cobra.Command{
		Use:   "parse",
		Short: "Parse YAML files (profiles, collections, base, manipulation) and store them in the database",
		Run:   pCmd,
	}
	emptyGrah         = "keep-upstream"
	emptyMinio        = false
	emptyGraphOptions = []string{
		"keep-upstream",
		"all",
		"none",
	}
)

func init() {
	rootCmd.AddCommand(parseCmd)
	// Here you will define your flags and configuration settings.
	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// parseCmd.PersistentFlags().String("foo", "", "A help for foo")
	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	emptyUsage := fmt.Sprintf("Empty the graph before parsing (%s)", strings.Join(emptyGraphOptions, "|"))
	parseCmd.PersistentFlags().StringVar(&emptyGrah, "empty-graph", "none", emptyUsage)
	parseCmd.PersistentFlags().BoolVar(&emptyMinio, "empty-minio", false, "Empty the object store before parsing")
}

func pCmd(cmd *cobra.Command, args []string) {
	config.SetCommandValues(cmd)
}
