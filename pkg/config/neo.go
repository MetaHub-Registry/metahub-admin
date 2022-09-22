package config

const (
	NeoURL = "bolt://localhost:7687"
)

type NeoConfig struct {
	DbName string
	User   string
	Passwd string
	URL    string
}
