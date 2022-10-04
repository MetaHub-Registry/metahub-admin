package transform

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (t Transformer) CollectionToSqlite() (err error) {
	cols, err := t.Neo.GetAllCollections()
	if err != nil {
		return
	}
	for idx, c := range cols {
		err = t.SQlite.Collection.Insert(&c)
		if err != nil {
			err = errors.Wrap(err, "t.SQlite.Collection.Insert(&c)")
			logrus.Error(err)
			continue
		}
		// iterate over ManifestLists
		for i, ml := range c.ManifestLists {
			// insert ManifestList into SQLite
			err = t.SQlite.ManifestList.Insert(&ml)
			if err != nil {
				err = errors.Wrap(err, "t.SQlite.ManifestList.Insert(&ml)")
				logrus.Error(err)
				continue
			}
			c.ManifestLists[i] = ml
			// Insert ManifestList-Collection relation into SQLite
			err = t.SQlite.Collection.InsertXTable(c.SqlID, ml.SqlID)
			if err != nil {
				err = errors.Wrap(err, "t.SQlite.InsertXTable.Insert(c.SqlID, ml.SqlID)")
				logrus.Error(err)
				continue
			}
		}
		cols[idx] = c
	}
	return
}
