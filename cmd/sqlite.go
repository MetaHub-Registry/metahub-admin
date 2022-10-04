/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/metahub-registry/metahub-admin/pkg/backends/minio"
	"github.com/metahub-registry/metahub-admin/pkg/backends/neo"
	"github.com/metahub-registry/metahub-admin/pkg/backends/sqlite"
	"github.com/metahub-registry/metahub-admin/pkg/backends/transform"
	"github.com/metahub-registry/metahub-admin/pkg/config"
)

// sqliteCmd represents the sqlite command
var sqliteCmd = &cobra.Command{
	Use:   "sqlite",
	Short: "Command to dump the graph DB into an sqlite database",
	Long:  `The sqlite database is more compact and is used to drive static API endpoints`,
	Run:   Cmd,
}

func init() {
	rootCmd.AddCommand(sqliteCmd)

	// Here you will define your flags and configuration settings.
	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	//// SQLite
	sqliteCmd.PersistentFlags().String("file", "./metahub.db", "The database file to use - will be overwritten")
	//// Neo4j
	sqliteCmd.PersistentFlags().String("neo-url", config.NeoURL, "NEO4j database url")
	viper.BindPFlag("neo_url", rootCmd.Flags().Lookup("neo-url"))
	sqliteCmd.PersistentFlags().String("neo-db", config.NeoDB, "NEO4j database name")
	viper.BindPFlag("neo_db", rootCmd.Flags().Lookup("neo-db"))
	sqliteCmd.PersistentFlags().String("neo-user", config.NeoUser, "NEO4j database username")
	viper.BindPFlag("neo_user", rootCmd.Flags().Lookup("neo-user"))
	sqliteCmd.PersistentFlags().String("neo-pass", config.NeoPass, "NEO4j database password")
	viper.BindPFlag("neo_pass", rootCmd.Flags().Lookup("neo-pass"))

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// sqliteCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func Cmd(cmd *cobra.Command, args []string) {
	config.SetCommandValues(cmd)
	cfgFile, _ := cmd.Flags().GetString("config")
	dbFile, _ := cmd.Flags().GetString("file")
	sq, err := sqlite.New(dbFile)
	defer sq.Close()
	checkErr(err, "while sqlite.New")
	m, err := minio.NewMinio(cfgFile)
	checkErr(err, "while NewMinio")
	err = m.BucketCheck()
	checkErr(err, "while BucketCheck")
	neo, err := neo.New(cfgFile)
	checkErr(err, "while neo.New")
	err = neo.Setup()
	checkErr(err, "while neo.Setup")
	t, err := transform.New(sq, neo, m)
	checkErr(err, "while transform.New")
	err = t.NeoToSQLite()
	checkErr(err, "while t.ToSQLite")
	logrus.Info("END sqlite called")
}

func checkErr(err error, msg string) {
	if err != nil {
		if len(msg) > 0 {
			err = errors.Wrap(err, msg)
		}
		panic(err)
	}
}
