package cmd

import (
	"encoding/json"
	"testing"

	"github.com/metahub-registry/metahub-core/pkg/types"
	yamlCfg "github.com/metahub-registry/metahub-core/pkg/yaml/config"
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

// minicpm5Model mirrors community-edition/models/qnib/minicpm5/latest.yaml:
// default config + a frame-pinned override.
func minicpm5Model() yamlCfg.Model {
	return yamlCfg.Model{
		Name:    "qnib/minicpm5",
		Runtime: "qnib/llama-server",
		Quants:  []yamlCfg.ModelQuant{{Tag: "q8-0", Source: "harbor.example.com/library/models/minicpm5:q8-0"}},
		Config: yamlCfg.ModelConfig{
			Default: map[string]any{"ctx_size": 32768, "kv_cache_type": "q8_0"},
			Overrides: []yamlCfg.ModelConfigOverride{
				{Profile: "frame", Config: map[string]any{"ctx_size": 65536}},
			},
		},
	}
}

func TestEffectiveConfigBytesDefault(t *testing.T) {
	b, err := effectiveConfigBytes(minicpm5Model(), "bazzite")
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got["ctx_size"].(float64) != 32768 || got["kv_cache_type"] != "q8_0" {
		t.Fatalf("unexpected default config: %s", b)
	}
}

func TestEffectiveConfigBytesProfileOverride(t *testing.T) {
	b, err := effectiveConfigBytes(minicpm5Model(), "frame")
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	// frame override replaces ctx_size but keeps the default kv_cache_type
	if got["ctx_size"].(float64) != 65536 || got["kv_cache_type"] != "q8_0" {
		t.Fatalf("expected merged override config, got: %s", b)
	}
}

func TestEffectiveConfigBytesEmpty(t *testing.T) {
	m := minicpm5Model()
	m.Config = yamlCfg.ModelConfig{} // nothing declared → nil (repackage default kicks in)
	b, err := effectiveConfigBytes(m, "frame")
	if err != nil {
		t.Fatal(err)
	}
	if b != nil {
		t.Fatalf("expected nil for empty config, got %s", b)
	}
}

func TestEffectiveConfigBytesAutoUnsupported(t *testing.T) {
	m := minicpm5Model()
	m.Config = yamlCfg.ModelConfig{Default: "auto"}
	if _, err := effectiveConfigBytes(m, "frame"); err == nil {
		t.Fatal("expected error for config.default \"auto\" (not implemented)")
	}
}
