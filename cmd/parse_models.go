package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/metahub-registry/metahub-admin/pkg/config"
	"github.com/metahub-registry/metahub-core/pkg/backends/magazine"
	"github.com/metahub-registry/metahub-core/pkg/types"
	yamlCfg "github.com/metahub-registry/metahub-core/pkg/yaml/config"
)

// defaultModelArtifactType is the ORAS artifactType the cached model sources use
// (mirrors test-repackage.sh's ARTIFACT_TYPE).
const defaultModelArtifactType = "application/vnd.qnib.model.artifact"

var (
	pmArtifactType string
	pmRegistryURL  string
	pmRegistryUser string
	pmRegistryPass string
	pmRegistryTLS  bool
)

// parseModelsCmd ingests declarative CE models/ entries: for each model it
// repackages every quant onto every model-serving profile via the shared
// repackage core. ORAS sources must already be cached in metahub (as `parse
// collections` also requires) — the external quants[].source auto-cache is a
// follow-up.
var parseModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Ingest CE models/ entries: repackage each quant per model-serving profile (sources must be cached first)",
	Run:   pModelsCmd,
}

func init() {
	parseCmd.AddCommand(parseModelsCmd)
	f := parseModelsCmd.Flags()
	f.StringVar(&pmArtifactType, "artifact-type", defaultModelArtifactType, "ORAS artifactType of the cached model sources")
	f.StringVar(&pmRegistryURL, "registry-url", "", "metahub OCI registry host to push repackaged images to")
	f.StringVar(&pmRegistryUser, "registry-user", "", "registry push user")
	f.StringVar(&pmRegistryPass, "registry-pass", "", "registry push password/token")
	f.BoolVar(&pmRegistryTLS, "registry-tls", true, "use TLS for the registry push")
}

// modelArtifactRepo is the in-metahub cache location of a quant's ORAS source:
// models/<name>/<quant>. A future caching step pulls quants[].source (an external
// ref) into this location; `parse models` assumes it is already present.
func modelArtifactRepo(modelName, quant string) string {
	return "models/" + modelName + "/" + quant
}

// modelServingProfiles filters profiles down to those that can serve models — i.e.
// those carrying a hardware.gpu block (Hardware.IsModelServing).
func modelServingProfiles(prfs []types.Profile) []types.Profile {
	serving := make([]types.Profile, 0, len(prfs))
	for _, p := range prfs {
		if p.Hardware.IsModelServing() {
			serving = append(serving, p)
		}
	}
	return serving
}

// effectiveConfigBytes renders the model's effective runtime config for one
// serving profile: Model.Config.Default merged with any profile-pinned
// Config.Overrides entry (Profile == profile). Rule-based overrides are
// deferred to the pull path; profile-pinned ones are applied here so the
// synthesised config artefact carries the CE-declared per-profile config.
// Returns nil when the rendered config is empty — the caller then leaves
// rpConfigFile unset so repackage falls back to its built-in default.
// A `default: auto` (derive-from-runtime) declaration is not yet supported
// (B5 render is future work) and yields nil with a warning-worthy error.
func effectiveConfigBytes(m yamlCfg.Model, profile string) ([]byte, error) {
	cfg := map[string]any{}
	switch d := m.Config.Default.(type) {
	case map[string]any:
		for k, v := range d {
			cfg[k] = v
		}
	case string:
		if d != "" && d != "auto" {
			return nil, fmt.Errorf("model %q: unsupported config.default %q (map or \"auto\")", m.Name, d)
		}
		if d == "auto" {
			return nil, fmt.Errorf("model %q: config.default \"auto\" (derive) not implemented yet", m.Name)
		}
	case nil:
		// no default declared
	default:
		return nil, fmt.Errorf("model %q: unsupported config.default type %T", m.Name, m.Config.Default)
	}
	for _, o := range m.Config.Overrides {
		if o.Profile == profile {
			for k, v := range o.Config {
				cfg[k] = v
			}
		}
	}
	if len(cfg) == 0 {
		return nil, nil
	}
	return json.Marshal(cfg)
}

// writeTempConfig writes configBytes to a temp JSON file for rpConfigFile and
// returns the path (the caller removes it after the run).
func writeTempConfig(cfgBytes []byte) (string, error) {
	f, err := os.CreateTemp("", "metahub-model-config-*.json")
	if err != nil {
		return "", err
	}
	if _, err := f.Write(cfgBytes); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), f.Close()
}

// loadModels walks the given directories and returns the CE models found,
// keyed like FindModels. A directory that fails to scan is logged and skipped.
func loadModels(ctx context.Context, dirs []string, l *logrus.Entry) map[string]yamlCfg.Model {
	models := map[string]yamlCfg.Model{}
	for _, a := range dirs {
		ms, err := yamlCfg.FindModels(ctx, a)
		if err != nil {
			l.Error(err)
			continue
		}
		for k, m := range ms {
			models[k] = m
		}
	}
	return models
}

// stageTempConfig renders the model's effective config for one profile into a
// temp file (tracked in tmpConfigs for cleanup) and returns the path, or ""
// when nothing was rendered (repackage then uses its built-in default).
func stageTempConfig(m yamlCfg.Model, profile string, tmpConfigs *[]string, l *logrus.Entry) string {
	cfgBytes, err := effectiveConfigBytes(m, profile)
	if err != nil {
		l.Warnf("  config render for profile %s failed (%v) — using repackage default", profile, err)
		return ""
	}
	if len(cfgBytes) == 0 {
		return ""
	}
	path, err := writeTempConfig(cfgBytes)
	if err != nil {
		l.Warnf("  temp config write failed (%v) — using repackage default", err)
		return ""
	}
	*tmpConfigs = append(*tmpConfigs, path)
	l.Infof("  config for profile %s: %s", profile, string(cfgBytes))
	return path
}

// repackageModelQuant drives one (quant × profile) repackage via the shared
// repackage core, wired from the parse-models flags.
func repackageModelQuant(ctx context.Context, mag magazine.Magazine, m yamlCfg.Model, q yamlCfg.ModelQuant, prf types.Profile, configFile string, l *logrus.Entry) {
	rpArtifactRepo = modelArtifactRepo(m.Name, q.Tag)
	rpArtifactType = pmArtifactType
	rpName = m.Name
	rpQuantTag = q.Tag
	rpProfile = prf.Label
	rpRuntime = m.Runtime
	rpTargetPath = ""
	rpRegistryURL = pmRegistryURL
	rpRegistryUser = pmRegistryUser
	rpRegistryPass = pmRegistryPass
	rpRegistryTLS = pmRegistryTLS
	rpSkipManip = false
	rpConfigFile = configFile
	l.Infof("  repackage %s:%s → profile %s (artifact %s, runtime %s)", m.Name, q.Tag, prf.Label, rpArtifactRepo, m.Runtime)
	repackageOnce(ctx, mag)
}

func pModelsCmd(cmd *cobra.Command, args []string) {
	l := logrus.WithFields(logrus.Fields{"package": "cmd/parse_models", "func": "pModelsCmd"})
	ctx := context.Background()
	config.SetCommandValues(cmd)
	if len(args) == 0 {
		l.Error("provide the directory to recursively search for models/ YAML files as the first argument")
		return
	}
	models := loadModels(ctx, args, l)

	mag, ctx, err := magazine.NewMagazine(cmd, MH_TABLES)
	if err != nil {
		l.Fatal(err)
	}

	allPrfs, err := mag.Database.ProfileGetAll(ctx)
	if err != nil {
		l.Fatalf("ProfileGetAll: %v", err)
	}
	serving := modelServingProfiles(allPrfs)
	if len(serving) == 0 {
		l.Warn("no model-serving profiles (with a hardware.gpu block) seeded; nothing to ingest. Run `parse profiles` on GPU profiles first.")
		return
	}

	tmpConfigs := make([]string, 0)
	defer func() {
		for _, p := range tmpConfigs {
			os.Remove(p)
		}
	}()
	for _, m := range models {
		l.Infof("# Model %s (runtime=%s, %d quants) → %d model-serving profile(s)", m.Name, m.Runtime, len(m.Quants), len(serving))
		for _, q := range m.Quants {
			for _, prf := range serving {
				// Honor the CE-declared per-profile config: render default +
				// profile-pinned overrides into a temp file consumed by
				// repackage's synthetic-config path (defaultConfigBytes).
				configFile := stageTempConfig(m, prf.Label, &tmpConfigs, l)
				repackageModelQuant(ctx, mag, m, q, prf, configFile, l)
			}
		}
	}
}
