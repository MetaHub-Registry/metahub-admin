package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/metahub-registry/metahub-admin/pkg/config"
)

var parseManifestCmd = &cobra.Command{
	Use:   "manifest",
	Short: "A brief description of your command",
	Run: func(cmd *cobra.Command, args []string) {
		config.SetCommandValues(cmd)
		neoUrl, _ := cmd.Flags().GetString("neo-url")
		fmt.Println("neoUrl: ", neoUrl)
	},
}

func init() {
	parseCmd.AddCommand(parseManifestCmd)
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// parseCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// parseCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
