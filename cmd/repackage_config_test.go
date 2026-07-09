package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfigBytes(t *testing.T) {
	orig := rpConfigFile
	defer func() { rpConfigFile = orig }()

	// No --config → a minimal default carrying the abstract config keys.
	rpConfigFile = ""
	b := defaultConfigBytes()
	if !strings.Contains(string(b), "ctx_size") || !strings.Contains(string(b), "kv_cache_type") {
		t.Fatalf("default config missing abstract keys: %s", b)
	}

	// --config <file> → that file's exact content.
	f := filepath.Join(t.TempDir(), "cfg.json")
	if err := os.WriteFile(f, []byte(`{"custom":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	rpConfigFile = f
	if got := string(defaultConfigBytes()); got != `{"custom":true}` {
		t.Fatalf("--config not read verbatim: %q", got)
	}
}
