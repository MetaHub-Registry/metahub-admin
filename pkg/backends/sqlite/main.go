package sqlite

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
)

var dbMh *sql.DB
var dbUsrMgmt *sql.DB

type Sqlite struct {
	// File is the path to the sqlite database file
	MhFile       string
	UsrFile      string
	Sql          *sql.DB
	User         User
	Token        Token
	Profile      Profile
	Config       Config
	Platform     Platform
	Filter       Filter
	Upstream     Upstream
	Manifest     Manifest
	Collection   Collection
	ManifestList ManifestList
}

func NewUsrBackend(usrFile string) (s Sqlite, err error) {
	return New("", usrFile)
}

func New(mhFile, usrFile string) (s Sqlite, err error) {
	s = Sqlite{
		MhFile:       mhFile,
		UsrFile:      usrFile,
		User:         User{},
		Token:        Token{},
		Config:       Config{},
		Profile:      Profile{},
		Filter:       Filter{},
		Platform:     Platform{},
		Upstream:     Upstream{},
		Manifest:     Manifest{},
		Collection:   Collection{},
		ManifestList: ManifestList{},
	}
	s.createDBs()
	return
}
func (s Sqlite) Close() {
	if s.MhFile != "" {
		dbMh.Close()
	}
	dbUsrMgmt.Close()
}

func (s Sqlite) createViews() (err error) {
	if s.MhFile != "" {
		if err = s.Collection.CreateViews(); err != nil {
			return
		}
		if err = s.Manifest.CreateViews(); err != nil {
			return
		}
	}
	if s.UsrFile != "" {
		return
	}
	return
}
func (s *Sqlite) createDBs() (err error) {
	if err := s.createUsrDB(); err != nil {
		return err
	}
	if s.MhFile != "" {
		err = s.createMhDB()
	}
	return
}

func (s *Sqlite) createUsrDB() (err error) {
	dbMh, err = sql.Open("sqlite3", s.UsrFile)
	checkErr(err, "while opening user-sqlite3")
	// Create tables
	err = s.User.CreateTables()
	checkErr(err, "while creating user-tables")
	return
}

func (s *Sqlite) createMhDB() (err error) {
	dbMh, err = sql.Open("sqlite3", s.MhFile)
	checkErr(err, "while opening metahub-sqlite3")
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
