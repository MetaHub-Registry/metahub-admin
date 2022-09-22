/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/metahub-registry/metahub-admin/pkg/config"
)

// parseCmd represents the parse command
var parseCmd = &cobra.Command{
	Use:   "parse",
	Short: "A brief description of your command",
	Run: func(cmd *cobra.Command, args []string) {
		config.SetCommandValues(cmd)
		neoUrl, _ := cmd.Flags().GetString("neo-url")
		fmt.Println("neoUrl: ", neoUrl)
	},
}

func init() {
	rootCmd.AddCommand(parseCmd)
	// Here you will define your flags and configuration settings.
	parseCmd.PersistentFlags().StringVar(&neoConfig.URL, "neo-url", config.NeoURL, fmt.Sprintf("NEO4j database url (default is '%s')", config.NeoURL))
	viper.BindPFlag("neo_url", rootCmd.Flags().Lookup("neo-url"))
	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// parseCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// parseCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
