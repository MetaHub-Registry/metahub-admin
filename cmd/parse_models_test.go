package cmd

import (
	"testing"

	"github.com/metahub-registry/metahub-core/pkg/types"
)

func TestModelArtifactRepo(t *testing.T) {
	if got := modelArtifactRepo("qnib/minicpm5", "q8-0"); got != "models/qnib/minicpm5/q8-0" {
		t.Fatalf("modelArtifactRepo = %q", got)
	}
}

func TestModelServingProfiles(t *testing.T) {
	prfs := []types.Profile{
		{Label: "frame", Hardware: &types.Hardware{GPU: &types.HardwareGPU{Vendor: "amd", VRAMGB: 124}}},
		{Label: "cpu#zen2"}, // no hardware block → not model-serving
		{Label: "bazzite", Hardware: &types.Hardware{GPU: &types.HardwareGPU{Vendor: "nvidia", VRAMGB: 12}}},
		{Label: "nogpu", Hardware: &types.Hardware{System: &types.HardwareSystem{RAMGB: 64}}}, // hardware but no GPU
	}
	serving := modelServingProfiles(prfs)
	if len(serving) != 2 {
		t.Fatalf("expected 2 model-serving profiles, got %d (%+v)", len(serving), serving)
	}
	for _, p := range serving {
		if p.Label == "cpu#zen2" || p.Label == "nogpu" {
			t.Fatalf("%s must not be model-serving", p.Label)
		}
	}
}
