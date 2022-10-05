package neo

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
	"github.com/sirupsen/logrus"
	yamlCfg "github.com/metahub-registry/metahub-admin/pkg/yaml/config"
	"github.com/metahub-registry/metahub-api/pkg/types"
)

func (n Neo) GetCollection(col yamlCfg.ManifestCollection) (colId int64, success bool, err error) {
	q := `MATCH (c:Collection) WHERE c.repo = "%s" AND c.tag = "%s" RETURN id(c) AS collectionID`
	query := fmt.Sprintf(q, col.Name, col.Tag)
	session := n.neoDriver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()
	result, err := session.Run(query, map[string]interface{}{})
	if err != nil {
		logrus.Errorf("Failed to run query: %s", err)
		return
	}
	if result.Next() {
		colId = result.Record().GetByIndex(0).(int64)
		success = true
	}
	return
}

func (n Neo) GetAllCollections() (collections []types.Collection, err error) {
	session := n.neoDriver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()
	//q := `MATCH (c:Collection)--(m:Manifest)--(p:Platform)--(f:Filter)--(pro:Profile)`
	//q += ` MATCH (c)<-[:SERVES]-(ml:ManifestList)-[:IMPLEMENTS]->(f)`
	q := `MATCH (c:Collection)<-[:SERVES]-(ml:ManifestList)-[:IMPLEMENTS]->(f) WHERE ml.label IS NOT NULL`
	q += ` RETURN id(c) AS collectionID, id(ml) AS manifestListID, id(f) AS filterID,`
	q += ` c.repo AS collectionRepo, c.tag AS collectionTag,`
	q += ` ml.digest AS manifestListDigest, ml.mediaType AS manifestListMediaType, ml.size AS manifestListSize`
	input := map[string]interface{}{}
	result, err := session.Run(q, input)
	if err != nil {
		return
	}
	cols := map[int64]types.Collection{}
	collections = make([]types.Collection, 0)
	for result.Next() {
		c, e := CollectionFromRecord(result)
		if e != nil {
			err = e
			return
		}
		ml, e := ManifestListFromRecord(result)
		if e != nil {
			err = e
			return
		}
		// Get all manifests for this manifestList
		ml.Manifests, err = n.GetManifestsForManifestList(ml.NeoID)
		if err != nil {
			return
		}
		col, ok := cols[c.NeoID]
		if !ok {
			c.ManifestLists = append(c.ManifestLists, ml)
			cols[c.NeoID] = c
		} else {
			col.ManifestLists = append(cols[c.NeoID].ManifestLists, ml)
			cols[c.NeoID] = col
		}
	}
	for _, c := range cols {
		collections = append(collections, c)
	}
	return
}

func (n Neo) CreateCollection(col yamlCfg.ManifestCollection) (colId int64, err error) {
	q := `CREATE (c:Collection {repo: "%s", tag: "%s", label: "%s"}) RETURN id(c) AS collectionID`
	label := fmt.Sprintf("%s:%s", col.Name, col.Tag)
	query := fmt.Sprintf(q, col.Name, col.Tag, label)
	session := n.neoDriver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()
	result, err := session.Run(query, map[string]interface{}{})
	if err != nil {
		logrus.Errorf("failed to create collection: %v", err)
		return
	}
	if result.Next() {
		v, ok := result.Record().Get("collectionID")
		if !ok {
			err = fmt.Errorf("failed to get collectionID")
			return
		}
		colId = v.(int64)
	}
	return
}

func CollectionFromRecord(res neo4j.Result) (c types.Collection, err error) {
	r := res.Record()
	v, ok := r.Get("collectionID")
	if !ok {
		err = fmt.Errorf("no manifestListID found in record: %v", r)
		return
	}
	c.NeoID = v.(int64)
	v, ok = r.Get("collectionRepo")
	if !ok {
		err = fmt.Errorf("no collectionRepo found in record: %v", r)
		return
	}
	c.Repo = v.(string)
	v, ok = r.Get("collectionTag")
	if !ok {
		err = fmt.Errorf("no collectionTag found in record: %v", r)
		return
	}
	c.Tag = v.(string)
	c.ManifestLists = make([]types.ManifestList, 0)
	return
}

func ManifestListFromRecord(res neo4j.Result) (ml types.ManifestList, err error) {
	r := res.Record()
	v, ok := r.Get("manifestListID")
	if !ok {
		err = fmt.Errorf("no manifestListID found in record: %v", r)
		return
	}
	ml.NeoID = v.(int64)
	v, ok = r.Get("manifestListDigest")
	if !ok {
		err = fmt.Errorf("no manifestListDigest found in record: %v", r)
		return
	}
	ml.Digest = v.(string)
	v, ok = r.Get("manifestListMediaType")
	if !ok {
		err = fmt.Errorf("no manifestListMediaType found in record: %v", r)
		return
	}
	ml.MediaType = v.(string)
	v, ok = r.Get("manifestListSize")
	if !ok {
		err = fmt.Errorf("no manifestListSize found in record: %v", r)
		return
	}
	ml.Size = v.(int64)
	ml.Manifests = make([]types.Manifest, 0)
	return
}

func (n Neo) LinkCollectionToManifest(ctx context.Context, colID, maniID int64) (err error) {
	q := `MATCH (c:Collection), (m:Manifest) WHERE id(c) = %d AND id(m) = %d`
	q += ` MERGE (c)-[:UPSTREAM]->(m)`
	query := fmt.Sprintf(q, colID, maniID)
	session := n.neoDriver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()
	_, err = session.Run(query, map[string]interface{}{})
	if err != nil {
		logrus.Errorf("failed to create link collection-manifest: %v", err)
		return
	}
	return
}
