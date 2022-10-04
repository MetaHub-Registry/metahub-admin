package config

import (
	"path/filepath"

	"github.com/metahub-registry/metahub-admin/pkg/upstream"
	"gitlab.com/qnib-golang/metahub/pkg/lib/utils"
)

type Manifest struct {
	Registry string   `yaml:"registry"`
	Image    string   `yaml:"image"`
	Platform Platform `yaml:"platform"`
}

func (m Manifest) ToUpstreamTriple() (u upstream.UpstreamTriple, err error) {
	reg, repo, tag, err := utils.SplitImageName(filepath.Join(m.Registry, m.Image))
	if err != nil {
		return
	}
	u = upstream.UpstreamTriple{
		Reg:  reg,
		Repo: repo,
		Tag:  tag,
	}
	return
}
