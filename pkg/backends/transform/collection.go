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
		logrus.Infof("Collection %d: %s:%s", idx, c.Repo, c.Tag)
		col, err := t.SQlite.GetInsertCollectionByRepoTag(c.Repo, c.Tag)
		if err != nil {
			err = errors.Wrap(err, "t.SQlite.GetInsertCollectionByRepoTag")
			logrus.Error(err)
			continue
		}
		col.ManifestLists = c.ManifestLists
		/*
			err = t.Neo.AddMLtoCollection(&col)
			if err != nil {
				err = errors.Wrap(err, "t.Mag.AddMLtoCollection")
				logrus.Error(err)
				continue
			}
		*/
		// iterate over ManifestLists
		for i, ml := range col.ManifestLists {
			logrus.Infof("ManifestList %d: %s", i, ml.Digest)
			// insert ManifestList into SQLite
			err = t.SQlite.InsertManifestList(&ml)
			if err != nil {
				err = errors.Wrap(err, "t.SQlite.ManifestList.Insert(&ml)")
				logrus.Error(err)
				continue
			}
			col.ManifestLists[i] = ml
			// Insert ManifestList-Collection relation into SQLite
			err = t.SQlite.Collection.InsertXTable(col.ID, col.ColTag.ID, ml.ID)
			if err != nil {
				err = errors.Wrap(err, "t.SQlite.InsertXTable.Insert(c.ID, ml.ID)")
				logrus.Error(err)
				continue
			}
		}
		cols[idx] = col
	}
	return
}
