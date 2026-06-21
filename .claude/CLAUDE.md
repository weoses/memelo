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