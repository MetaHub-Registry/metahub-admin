package config

const (
	NeoURL  = "bolt://localhost:7687"
	NeoDB   = "neo4j"
	NeoUser = "neo4j"
	NeoPass = "test"
)

type NeoConfig struct {
	DbName string
	User   string
	Passwd string
	URL    string
}
