package magazine

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/metahub-registry/metahub-admin/pkg/backends/minio"
	"github.com/metahub-registry/metahub-admin/pkg/backends/neo"
	"github.com/metahub-registry/metahub-admin/pkg/backends/sqlite"
)

type Magazine struct {
	Neo    neo.Neo
	Minio  minio.Minio
	SQlite sqlite.Sqlite
}

func NewMagazine(cmd *cobra.Command) (m Magazine, err error) {
	cfgFile, _ := cmd.Flags().GetString("config")
	min, err := minio.NewMinio(cfgFile)
	if err != nil {
		logrus.Fatal(err)
	}
	err = min.BucketCheck()
	if err != nil {
		logrus.Fatal(err)
	}
	m = Magazine{
		Minio: min,
	}

	return
}
