package sqlite

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	apiSqlite "github.com/metahub-registry/metahub-api/pkg/backends/sqlite"
)

var db *sql.DB

const dbTimeout = time.Second * 3

type Sqlite struct {
	// File is the path to the sqlite database file
	File         string
	Profile      apiSqlite.Profile
	Config       apiSqlite.Config
	Platform     apiSqlite.Platform
	Filter       apiSqlite.Filter
	Upstream     apiSqlite.Upstream
	Manifest     apiSqlite.Manifest
	Collection   apiSqlite.Collection
	ManifestList apiSqlite.ManifestList
}

func New(dbFile string) (s Sqlite, err error) {
	s = Sqlite{
		File:         dbFile,
		Config:       apiSqlite.Config{},
		Profile:      apiSqlite.Profile{},
		Filter:       apiSqlite.Filter{},
		Platform:     apiSqlite.Platform{},
		Upstream:     apiSqlite.Upstream{},
		Manifest:     apiSqlite.Manifest{},
		Collection:   apiSqlite.Collection{},
		ManifestList: apiSqlite.ManifestList{},
	}
	s.createDB()
	return
}
func (s Sqlite) Close() {
	db.Close()
}

func (s Sqlite) createViews() (err error) {
	if err = s.Collection.CreateViews(); err != nil {
		return
	}
	if err = s.Manifest.CreateViews(); err != nil {
		return
	}
	return
}
func (s Sqlite) createDB() (err error) {
	db, err = sql.Open("sqlite3", s.File)
	checkErr(err, "while opening sqlite3")
	// Create tables
	err = s.Profile.CreateTables()
	checkErr(err, "")
	err = s.Filter.CreateTables()
	checkErr(err, "")
	err = s.Platform.CreateTables()
	checkErr(err, "")
	err = s.Config.CreateTables()
	checkErr(err, "")
	err = s.Upstream.CreateTables()
	checkErr(err, "")
	err = s.Manifest.CreateTables()
	checkErr(err, "")
	err = s.Collection.CreateTables()
	checkErr(err, "")
	err = s.ManifestList.CreateTables()
	checkErr(err, "")
	return
}

func checkErr(err error, msg string) {
	if err != nil {
		if len(msg) > 0 {
			err = errors.Wrap(err, msg)
		}
		panic(err)
	}
}
