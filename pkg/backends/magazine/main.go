package magazine

import (
	"github.com/pkg/errors"
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

func NewMagazine(cmd *cobra.Command, tables string) (m Magazine, err error) {
	cfgFile, _ := cmd.Flags().GetString("config")
	min, err := minio.NewMinio(cfgFile)
	if err != nil {
		err = errors.Wrapf(err, "failed to create minio backend")
		logrus.Fatal(err)
	}
	err = min.BucketCheck()
	if err != nil {
		err = errors.Wrapf(err, "failed to execute BucketCheck")
		logrus.Fatal(err)
	}
	// GraphDB
	neo, err := neo.New(cfgFile)
	if err != nil {
		err = errors.Wrapf(err, "failed to create Neo backend")
		logrus.Fatal(err)
	}
	neo.Setup()
	m = Magazine{
		Minio: min,
		Neo:   neo,
	}

	// SQLite
	mhSqlFile, _ := cmd.Flags().GetString("metahub-db")
	sqlite, err := sqlite.New(mhSqlFile, tables)
	if err != nil {
		err = errors.Wrapf(err, "failed to create sqlite backend")
		logrus.Fatal(err)
	}
	m.SQlite = sqlite
	return
}
