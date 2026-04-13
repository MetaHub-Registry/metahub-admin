package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
)

type Version struct {
	Major int `json:"major"`
	Minor int `json:"minor"`
	Patch int `json:"patch"`
	Dev   int `json:"dev,omitempty"`
}

func NewVersion(version string) (v Version, err error) {
	// Split off optional -devN suffix
	base := version
	var dev int
	if idx := strings.Index(version, "-dev"); idx != -1 {
		dev, err = strconv.Atoi(version[idx+4:])
		if err != nil {
			return v, fmt.Errorf("invalid dev number in version %q: %w", version, err)
		}
		base = version[:idx]
	}
	vSplit := strings.Split(base, ".")
	if len(vSplit) < 3 {
		return v, fmt.Errorf("invalid version format %q: expected major.minor.patch", version)
	}
	v = Version{
		Major: 0,
		Minor: 0,
		Patch: 0,
		Dev:   dev,
	}
	if major, e := strconv.Atoi(vSplit[0]); e == nil {
		v.Major = major
	} else {
		return v, e
	}
	if minor, e := strconv.Atoi(vSplit[1]); e == nil {
		v.Minor = minor
	} else {
		return v, e
	}
	if patch, e := strconv.Atoi(vSplit[2]); e == nil {
		v.Patch = patch
	} else {
		return v, e
	}
	return v, nil
}

func (v Version) ToString() string {
	if v.Dev > 0 {
		return fmt.Sprintf("%d.%d.%d-dev%d", v.Major, v.Minor, v.Patch, v.Dev)
	}
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}
func (v Version) ToJson() string {
	b, err := json.Marshal(v) //, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	return string(b)
}
