# model-config-injection Specification

## Purpose
TBD - created by archiving change oras-model-config-injection. Update Purpose after archive.
## Requirements
### Requirement: Config blob identified by a dedicated MediaType
`metahub-admin repackage` SHALL identify a configuration blob — carried alongside
the quantized model blob(s) in the pushed ORAS artifact and distinguished by the
dedicated MediaType `application/vnd.qnib.model.config.v1+json` — and SHALL split
the artifact's blobs into model blobs and the config blob so that neither is used
in place of the other. Repackage SHALL treat the config blob's contents as
**opaque**: it places the bytes; it does not parse or generate them.

#### Scenario: Config blob separated from model blobs by MediaType
- **WHEN** an artifact carries GGUF model blob(s) and one blob with MediaType
  `application/vnd.qnib.model.config.v1+json`
- **THEN** repackage SHALL route the model blob(s) to the model layer and the
  config blob to the config layer
- **AND** the model-image manifest SHALL NOT reference the config blob as a model
  layer, and the config layer SHALL NOT contain a model blob

#### Scenario: Model-only push is unchanged
- **WHEN** an artifact carries no blob of the config MediaType
- **THEN** repackage SHALL behave exactly as today (model layer + manipulation, no
  config)

### Requirement: Config repackaged into a per-profile config layer
When a config blob is present, `repackage` SHALL wrap it in a deterministic
uncompressed-tar **config layer** placing the file(s) at `/model/<name>/config/`
(the config analogue of the model layer's `/model/<name>/`), and SHALL attach that
config layer to the push `--profile` via the manipulation graph, so a pull under
that profile injects the model layer **and** the config layer through the existing
manipulation apply path — with no new pull-path logic.

#### Scenario: Config injected under its push profile
- **WHEN** `(model + config)` is pushed and repackaged with `--profile vram#12gb`
- **THEN** Neo4j SHALL hold the config layer attached to `vram#12gb`
- **AND** a pull of a base image under `vram#12gb` SHALL yield a served image
  carrying both the model file at `/model/<name>/…` and the config file at
  `/model/<name>/config/…`

### Requirement: Config is profile-specific
The config injected under a profile SHALL be the config the publisher pushed for
that profile: one `(model + config)` bundle is pushed per profile, `repackage`
attaches each config to that push's `--profile`, and configs SHALL be independent
across profiles. metahub SHALL NOT derive, size, or share config across profiles.

#### Scenario: 12gb and 128gb receive their own config
- **WHEN** `(model + config-A)` is pushed under `vram#12gb` and
  `(model + config-B)` under `vram#128gb`
- **THEN** a pull under `vram#12gb` SHALL inject `config-A` and a pull under
  `vram#128gb` SHALL inject `config-B`

