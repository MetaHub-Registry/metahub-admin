package transform

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (t Transformer) ManifestToSqlite() (err error) {
	/// 1. get all (Manifests)-(Upstream)-(Platform) nodes related to each other
	manis, err := t.Neo.GetAllManifests()
	if err != nil {
		return
	}
	for idx, m := range manis {
		logrus.Infof("Manifest %d: %s:%s", idx, m.Upstream.Repo, m.Upstream.Tag)
		// 1.a insert all configs
		cfg := m.Config
		err = t.SQlite.Config.Insert(&cfg)
		if err != nil {
			err = errors.Wrap(err, "t.SQlite.Config.Insert(&m.Config)")
			logrus.Error(err)
			continue
		}
		m.Config = cfg
		// 1.b insert Upstream into SQLite
		up := m.Upstream
		err = t.SQlite.Upstream.Insert(&up)
		if err != nil {
			err = errors.Wrap(err, "t.SQlite.Upstream.Insert(&m.Upstream)")
			logrus.Error(err)
			continue
		}
		m.Upstream = up
		// 1.c insert Platform into SQLite
		plat := m.Platform
		err = t.SQlite.Platform.Upsert(&plat)
		if err != nil {
			err = errors.Wrap(err, "Failed to insert platform: t.SQlite.Platform.Upsert(&m.Platform)")
			logrus.Error(err)
			continue
		}
		m.Platform = plat
		// Rewrite manifest in slice
		manis[idx] = m
		// 1.d insert Manifest into SQLite
		err = t.SQlite.Manifest.Insert(&m)
		if err != nil {
			logrus.Error(err)
			continue
		}
		/// 2. fetch all layers for manifest
		layers, err := t.Neo.GetLayersForManifest(m)
		if err != nil {
			logrus.Error(err)
			continue
		}
		for _, l := range layers {
			// 2.a insert layer into SQLite
			err = t.SQlite.Layer.Upsert(&l)
			if err != nil {
				logrus.Error(err)
				continue
			}
		}
	}
	return
}
