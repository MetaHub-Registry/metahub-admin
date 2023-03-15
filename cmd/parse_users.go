package cmd

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"

	"github.com/metahub-registry/metahub-admin/pkg/config"
	"github.com/metahub-registry/metahub-core/pkg/backends/magazine"
	"github.com/metahub-registry/metahub-core/pkg/types"
	yamlCfg "github.com/metahub-registry/metahub-core/pkg/yaml/config"
)

var parseUsersCmd = &cobra.Command{
	Use:   "users",
	Short: "Put users into the database",
	Run:   uParseCmd,
}

func init() {
	parseCmd.AddCommand(parseUsersCmd)
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// parseCmd.PersistentFlags().String("foo", "", "A help for foo")
	parseUsersCmd.Flags().String("user-db", "", "The user database file to use - will be overwritten")
	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// parseCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func uParseCmd(cmd *cobra.Command, args []string) {
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
	users, err := yamlCfg.FindUsers(ctx, args[0])
	if err != nil {
		logrus.Error(err)
	}

	mag, ctx, err := magazine.NewMagazine(cmd, "")
	if err != nil {
		logrus.Error(err)
	}
	defer mag.UserManagement.Close()
	for _, user := range users {
		u := types.User{
			Username:  user.Username,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Email:     user.Email,
			Role:      user.Role,
		}
		switch {
		case (user.Hash != "") && (user.Password != ""):
			logrus.Errorf("User '%s' has both 'hash' and 'password' set. Please remove one of them.", user.Username)
			continue
		case user.Hash != "":
			u.PasswordHash = user.Hash
		case user.Password != "":
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), 12)
			if err != nil {
				logrus.Errorf("Error while hashing password for user '%s': %s", user.Username, err)
				continue
			}
			u.PasswordHash = string(hashedPassword)
		}
		err := mag.InsertUser(ctx, u)
		if err != nil {
			logrus.Error(err)
		}

	}

}
