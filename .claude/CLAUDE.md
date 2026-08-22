# memelo codebase notes

## Project layout

- `storage-service/` — main Go service (media storage, extraction, search)
- `telegram-service/` — integration with telegram
- `proto/v1/` — protobuf definitions; regenerate with `buf generate` from repo root
- `gen/proto/v1/` — generated Go code, do not edit manually
- `common/` — shared utilities (helper, temp data, config)

## Service architecture

The service uses [Connect RPC](https://connectrpc.com/) (not raw gRPC). Handler interfaces are in `gen/proto/v1/v1connect/` and have plain signatures — no `connect.Request`/`connect.Response` wrappers on the handler side.

Dependency injection is done with `go.uber.org/fx`. New providers go in `storage-service/main.go`. Do not use one Provide call when create more than one object - try to use Result/Param Tags.

## Layering rule

**Proto/API types must not leak into the service layer.** The `api/` package owns all proto↔service mapping. Services define their own input/output structs (e.g. `RecomputeParams`, `RecomputeState`).

**Service/database types must not be returned by the API layer.** The `api/` package defines its own response DTOs (e.g. `MemeResponse`, `SearchResponse`) and maps from service types before serializing. Handlers must never pass service structs directly to `c.JSON`.

## Media extraction pipeline

Pipeline steps live in `storage-service/service/EP*.go`, named by position (e.g. `EP00`, `EP10`, …). Steps are registered in `main.go` with `fx` group tag `pipeline_steps` and sorted by `GetPos()` inside `NewImageMetadataExtractService`.

`MetadataExtractService.Extract` takes a `MetadataInputContext` which carries:
- `ComputeHash / ComputeExtractor / ComputeEmbedding` — each step checks its own flag and skips if false and a seeded value already exists
- `CheckDuplicates` — gates EP10 and EP40 duplicate checks
- `SeedData *MetadataPipelineContext` — pre-populate the context from existing metadata so skipped steps leave valid values for downstream steps

For new meme creation (`MemeCrudService`) all compute flags are `true`. For recompute jobs the flags come from the API request.

## Async recompute job system

- `service.RecomputeService.StartRecompute` launches a background goroutine and returns a job ID immediately.
- Job state is stored in `RecomputeJobStorage` (interface), default impl is a 1-hour TTL in-memory `gcache` (LFU, 256 entries).
- Worker concurrency is controlled by `maxRecomputeWorkers = 4` (buffered channel semaphore).
- Per-object errors are collected in `RecomputeJobState.Errors` and logged; they never abort the job. Only a pagination failure sets state to `FAILED`.
- `RecomputeJobState.Mu` is an exported `sync.RWMutex`; always hold it when reading or writing state fields.

## Elasticsearch / MetadataStorage

- All paginated queries use search-after with `entity.ElasticSortKey`.
- `QueryByRaw` accepts an arbitrary `map[string]interface{}` query and sends it as a raw JSON body to ES. Used by the recompute job to filter which objects to process.
- The typed ES client (`elasticsearch8.TypedClient`) is used everywhere except `QueryByRaw`, which uses `.Raw(io.Reader)` on the search builder.

## Resource closing

Temporary data (`temp.Data`, `temp.S3BackedData`) is backed by S3 objects and **must be closed** after use to avoid leaking storage. Use `helper.QuietClose(x, logger)` instead of `x.Close()` directly — it logs errors without panicking and is safe on nil values.

```go
raw, err := imageStorageService.Read(...)
defer helper.QuietClose(raw, r.slogger)
```

**Ownership rule:** whoever created the S3-backed resource is responsible for closing it. This applies at every level:

- A function or service that **receives** `temp.S3BackedData` as a parameter must **not** close it — the caller owns it and will close it.
- The storage service itself must **not** close data passed in from outside (e.g. raw media arriving via the API) — that data is owned by the caller and lives beyond the storage call.
- Only close resources you explicitly created (e.g. via `tmpDataService.WrapData`, `imageStorageService.Read`).

`MetadataPipelineContext` implements `io.Closer` and closes all its internal temp data — call `helper.QuietClose(pipelineResult, logger)` after you are done reading from it.

## Proto regeneration

```
buf generate   # run from repo root
```

## Terraform / Cloud Run deployment gotchas

Layout: `terraform/modules/` (reusable) + one dir per service, own GCS state
(`melo-terraform-state`, prefix `melo-<env>/<service>`), own dedicated SA.
Apply order: `global` → ffmpeg/youtube → storage → telegram/webapp (see
`terraform/README.md`). Secrets: sops+PGP per service
(`env/<env>/secrets.enc.env`), edit with `sops <file>`.

Non-obvious platform facts (cost real debugging time, not visible from code):

- **Cloud Run's default `*.run.app` URL is not computable** from project
  number/region/name — it's an opaque per-project+region hash, same for
  every service in that project+region, but not derivable in HCL. Don't
  hardcode it as a Terraform `local`; read a callee's real URI via
  `terraform_remote_state` (`data.terraform_remote_state.X.outputs.uri`)
  after that service's own state has been applied.
- **A resource can't reference its own computed attribute in its own
  config** (self-referencing URL problem: a service needing to know its own
  Cloud Run URL at boot, e.g. to register a webhook). Fix: deploy with a
  real-but-placeholder value first (must be an actually reachable HTTPS
  host — Telegram's `setWebhook` validates DNS/connectivity synchronously
  and rejects fake hosts like `foo.invalid` outright), then a
  `terraform_data` + `local-exec` provisioner (`triggers_replace =
  [timestamp()]`, runs every apply) patches the real value in via `gcloud
  run services update --update-env-vars=...` right after.
- **Never do cleanup-on-stop that assumes "stopping" means "service is
  going away"** in a Cloud Run app (e.g. unregistering an external
  webhook in an `OnStop`/shutdown hook). Every deploy retires the old
  revision's instance *after* the new one is already live, so its
  `OnStop` fires right after the new revision's `OnStart` already did the
  registration — deterministically undoing it, not a race. If a service's
  own external state (webhook, presence registration, etc.) depends on it
  being "up", it also needs `min_instances >= 1` — scale-to-zero hits the
  identical failure via idle shutdown, and nothing can wake it back up
  once its own external trigger is gone.
- **`scripts/entrypoint.sh` sources `/etc/secrets/*/*`** — a service with
  no secrets volume mounted leaves that glob unmatched, and under `set -e`
  the resulting `ls` failure aborts the script before the app ever starts.
  Guard with `[ -e "$f" ] || continue`.
- **GCP service account `account_id` has a 30-char cap.** Use short slugs
  ("storage", "ffmpeg") when generating one from `<env>-<slug>-service-account`.
- **docker-compose bind-mounts can mask a missing `COPY config.yaml` in a
  Dockerfile** — works locally, crashes on Cloud Run with "Config File not
  found". Check every service's `Dockerfile-*` actually copies its config
  when adding Cloud Run deployment for a service that previously only ran
  via docker-compose.
- Secret Manager secret-version creation is eventually consistent — Cloud
  Run can fail a deploy with "secret ... was not found" moments after
  Terraform just created that exact version. Just retry the apply.
- **A new Go config struct field is silently ignored by env-var overrides
  unless it also has a line in `config.yaml`.** `common/config/InitConfig.go`
  has a workaround loop (`for _, key := range viper.AllKeys() { ... }`)
  that re-materializes env-overridden values before `Unmarshal` — but
  `AllKeys()` only enumerates keys already known to Viper (i.e. present in
  the loaded config file). A field that exists only in the Go struct (e.g.
  `RequireGoogleIDToken` when it was first added) never appears in
  `AllKeys()`, so its env var is silently never applied — the field just
  keeps its Go zero value, no error, no log. This caused inter-service
  Google-IAM auth to look "wired up" (env var set correctly in Terraform,
  code compiled fine) while actually never sending an Authorization header
  at all (Cloud Run's own 403 was the only symptom, and it looks identical
  to an IAM misconfiguration). **Whenever adding a new config struct
  field that should be env-overridable, add a matching line to every
  affected `config.yaml`** (value doesn't matter, e.g. `false` — it just
  needs to exist so `AllKeys()` sees it).

Other operational notes: checking whether a CI build finished by polling
`ghcr.io`'s anonymous registry API works without any `gh`/GitHub auth at
all (`curl "https://ghcr.io/token?service=ghcr.io&scope=repository:weoses/<svc>:pull"`
→ use the token against `/v2/weoses/<svc>/tags/list`) — useful even if
`gh` is authenticated, since it's directly checking the actual deploy
artifact rather than CI run status. `gpg-agent`'s cache can expire
mid-session; `sops` decrypt then fails with "0 successful groups required,
got 0" — needs the user to unlock interactively (pinentry), can't be
relayed through non-interactive Bash. `scripts/create-tag.sh` requires a
fully clean git tree (staged+unstaged) and only creates the tag locally —
`git push origin <tag>` separately to actually trigger CI.