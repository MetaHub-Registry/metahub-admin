package upstream

import (
	"fmt"

	digestLib "github.com/opencontainers/go-digest"
)

type UpstreamTriple struct {
	Reg       string
	Repo      string
	Tag       string
	MediaType string
	Size      int64
	Digest    digestLib.Digest
}

func NewUpstreamTriple(k map[string]interface{}) (up UpstreamTriple, err error) {
	return NewUpstreamTripleBase(k["reg"].(string), k["repo"].(string), k["tag"].(string), k["mediaType"].(string), k["digest"].(string), k["size"].(int64))
}

func NewUpstreamTripleBase(reg, repo, tag, mediaType, digest string, size int64) (up UpstreamTriple, err error) {
	d, e := digestLib.Parse(digest)
	if e != nil {
		err = e
		return
	}
	up = UpstreamTriple{
		Reg:       reg,
		Repo:      repo,
		Tag:       tag,
		MediaType: mediaType,
		Size:      size,
		Digest:    d,
	}
	return
}

func (upt UpstreamTriple) GetLabel() string {
	return fmt.Sprintf("%s/%s:%s", upt.Reg, upt.Repo, upt.Tag)
}

func (upt UpstreamTriple) GetObjPath() string {
	// Storage path for manifest in Objectstore
	return fmt.Sprintf("%s/%s/%s", upt.Reg, upt.Repo, upt.Tag)
}
