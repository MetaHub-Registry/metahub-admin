/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/metahub-registry/metahub-admin/pkg/utils"
	mlver "github.com/metahub-registry/metahub-core/pkg/version"
)

var version = "2.3.0"
var jsonFlag bool

// parseCmd represents the parse command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints the version of the application",
	Run: func(cmd *cobra.Command, args []string) {
		maVer, e := utils.NewVersion(version)
		if e != nil {
			logrus.Fatalf("Failed to parse version: %v", e)
		}
		mlVer, e := utils.NewVersion(mlver.Version)
		if e != nil {
			logrus.Fatalf("Failed to parse version: %v", e)
		}
		if jsonFlag {
			ma := maVer.ToJson()
			fmt.Println(string(ma))
		} else {
			fmt.Printf("Admin:   v%s\n", maVer.ToString())
			fmt.Printf("Metalib: v%s\n", mlVer.ToString())
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	// Here you will define your flags and configuration settings.
	versionCmd.Flags().BoolVar(&jsonFlag, "json", false, "Prints the version in JSON format")
	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// parseCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// parseCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
