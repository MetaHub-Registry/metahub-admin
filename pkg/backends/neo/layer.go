package neo

import (
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
	"github.com/sirupsen/logrus"
	"github.com/metahub-registry/metahub-api/pkg/types"
	"gitlab.com/qnib-golang/metahub/pkg/lib/utils"
)

func (n Neo) GetLayer(layer types.Layer) (layerID int64, success bool, err error) {
	q := `MATCH (n:Layer) WHERE n.digest = "%s" AND n.mediaType = "%s" AND n.size = %d RETURN id(n) AS ID`
	query := fmt.Sprintf(q, layer.Digest, layer.MediaType, layer.Size)
	session := n.neoDriver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()
	result, err := session.Run(query, map[string]interface{}{})
	if err != nil {
		logrus.Errorf("Failed to run query: %s", err)
		return
	}
	if result.Next() {
		v, ok := result.Record().Get("ID")
		if !ok {
			err = fmt.Errorf("failed to get ID")
			return
		}
		layerID = v.(int64)
		success = true
	}
	return
}

func (n Neo) CreateLayer(layer types.Layer) (layerID int64, err error) {
	q := `CREATE (n:Layer {digest: "%s", mediaType: "%s", size: %d, label: "%s"}) RETURN id(n) AS ID`
	query := fmt.Sprintf(q, layer.Digest, layer.MediaType, layer.Size, utils.ShortenDigest(layer.Digest))
	session := n.neoDriver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()
	result, err := session.Run(query, map[string]interface{}{})
	if err != nil {
		logrus.Errorf("failed to create layer: %v", err)
		return
	}
	if result.Next() {
		v, ok := result.Record().Get("ID")
		if !ok {
			err = fmt.Errorf("failed to get ID")
			return
		}
		layerID = v.(int64)
	}
	return
}
