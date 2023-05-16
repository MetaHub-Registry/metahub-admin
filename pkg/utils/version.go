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
}

func NewVersion(version string) (v Version, err error) {
	vSplit := strings.Split(version, ".")
	v = Version{
		Major: 0,
		Minor: 0,
		Patch: 0,
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
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}
func (v Version) ToJson() string {
	b, err := json.Marshal(v) //, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	return string(b)
}
