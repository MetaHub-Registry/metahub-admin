package transform

import (
	"github.com/metahub-registry/metahub-core/pkg/backends/magazine"
	"github.com/metahub-registry/metahub-core/pkg/backends/minio"
	"github.com/metahub-registry/metahub-core/pkg/backends/neo"
	"github.com/metahub-registry/metahub-core/pkg/backends/sqlite"
)

type Transformer struct {
	Mag    magazine.Magazine
	SQlite sqlite.Sqlite
	Neo    neo.Neo
	Minio  minio.Minio
}

func New(sq sqlite.Sqlite, n neo.Neo, m minio.Minio) (t Transformer, err error) {
	mag := magazine.Magazine{
		Database:       n,
		Minio:          m,
		UserManagement: sq,
	}
	t = Transformer{
		Mag:    mag,
		SQlite: sq,
		Neo:    n,
		Minio:  m,
	}
	return
}

/*
func (t Transformer) NeoToSQLite() (err error) {
	// Profile and Filter to SQLite
	err = t.ProfilterToSqlite()
	if err != nil {
		return
	}
	// Manifest to SQLite
	err = t.ManifestToSqlite()
	if err != nil {
		return
	}
	// Collections and ManifestList to SQLite
	err = t.CollectionToSqlite()
	if err != nil {
		return
	}
	return
}
*/
