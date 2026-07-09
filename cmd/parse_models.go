package cmd

import (
	"context"

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

func pModelsCmd(cmd *cobra.Command, args []string) {
	l := logrus.WithFields(logrus.Fields{"package": "cmd/parse_models", "func": "pModelsCmd"})
	ctx := context.Background()
	config.SetCommandValues(cmd)
	if len(args) == 0 {
		l.Error("provide the directory to recursively search for models/ YAML files as the first argument")
		return
	}

	models := map[string]yamlCfg.Model{}
	for _, a := range args {
		ms, err := yamlCfg.FindModels(ctx, a)
		if err != nil {
			l.Error(err)
		}
		for k, m := range ms {
			models[k] = m
		}
	}

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

	for _, m := range models {
		l.Infof("# Model %s (runtime=%s, %d quants) → %d model-serving profile(s)", m.Name, m.Runtime, len(m.Quants), len(serving))
		for _, q := range m.Quants {
			for _, prf := range serving {
				// Drive the shared repackage core via its rp* parameters.
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
				l.Infof("  repackage %s:%s → profile %s (artifact %s, runtime %s)", m.Name, q.Tag, prf.Label, rpArtifactRepo, m.Runtime)
				repackageOnce(ctx, mag)
			}
		}
	}
}
