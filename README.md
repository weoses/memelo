# memelo

A meme management system with multi-modal search. Upload images/videos via Telegram or a web app, extract text and captions with an LLM, generate semantic embeddings, and search your collection by text, fuzzy match, or meaning.

### Integrations

Telegram - as inline bot [@memelobot](https://t.me/memelobot)  
WebUI - create, search, manage. Non-public. Still ugly, i am not a designer at all.

### Quick start

```bash
cp .env.example .env

# edit .env — at minimum set OPENROUTER_API_KEY
# (get one at https://openrouter.ai/settings/keys)

docker compose up -d --build
```

This starts everything locally: Elasticsearch, MinIO, Postgres, and all services below. No Google Cloud account needed by default.

Two services need extra credentials to actually do anything (both start fine without them, they just won't work):
- `telegram-service` needs a real `TELEGRAM_TOKEN` and a **public** `TELEGRAM_WEBHOOK_EXTERNALURL` (see [Telegram webhook](#telegram-webhook))
- `youtube-service` needs a real `YOUTUBE_PROVIDER_APIKEY` (see [YouTube downloads](#youtube-downloads))

| Service | Port | Purpose |
|---|---|---|
| `storage-service` | 7001 | Core service — media processing, extraction, search |
| `telegram-service` | 7002 | Telegram bot frontend |
| `webapp-service` | 7003 | Web UI (upload/search/browse) |
| `youtube-service` | 7004 | YouTube link → video download |
| `ffmpeg-service` | 7005 | Video conversion/thumbnail/slicing worker |
| `auth-service` | 7006 | User/permission/integration-identity authorization |
| `elasticsearch` | 9200 | Metadata + vector search |
| `minio` | 9000 | S3-compatible media storage |
| `postgres` | 5432 | `auth-service` user/permission data (`auth` database) |

### Modules

| Path | Description |
|---|---|
| `storage-service` | Core service — media processing, OCR/captioning, embeddings, search, export |
| `telegram-service` | Telegram bot frontend |
| `webapp-service` | Web UI backend + frontend (Vite/React) |
| `youtube-service` | Downloads YouTube videos for ingestion |
| `ffmpeg-service` | Ffmpeg-backed video conversion worker |
| `auth-service` | Users, permissions, and per-integration identities (Telegram, webapp login) |
| `common` | Shared config, logging, and helper utilities |
| `gen` | Generated protobuf/Connect RPC code (do not edit) |
| `proto` | Protocol buffer source definitions |

### Gemini vs OpenRouter

`storage-service` picks its LLM backend per config — `extractor-provider` (captions/OCR) and `embedder-provider` (search vectors) can each independently be `gemini` or `openrouter`.

- **Video embeddings**: Gemini embeds the whole video natively in one call. OpenRouter has no video embedding API — it grabs a single frame (via `ffmpeg-service`) and embeds that image instead. Gemini gives better video search quality; OpenRouter is simpler to set up.
- **API keys**: env vars `GEMINI_API_KEY` / `OPENROUTER_API_KEY` (root `.env`) feed both the extractor and embedder for that provider. No Vertex AI project or Google ADC file is required — both providers authenticate with a plain API key.
- Default is `openrouter` for local startup, since it needs nothing but a key.

### YouTube downloads

`youtube-service` doesn't use `yt-dlp` — it calls a third-party download API (default host `p.savenow.to`): submits a download job, polls until ready, then streams the result into MinIO.

Unfortunately, yt-dlp is quite unusable without JS runtime, cookies and other desktop-style stuff. Calling third-party api is simplier than export cookies from my PC to VM every day.

Config: `YOUTUBE_PROVIDER_APIKEY` (API key, root `.env`), `YOUTUBE_MAX_CONCURRENT_DOWNLOADS` (default 4). Video format/max duration are set in `youtube-service/config.yaml` (`youtube.VideoFormat`, `youtube.MaxDuration`).

## FFmpeg conversion

`ffmpeg-service` wraps three operations: convert-to-MP4, extract-thumbnail-frame, and slice-video-with-overlap (slicing uses `ffprobe` to get duration, then stream-copies segments — no re-encode).

Config (root `.env`): `FFMPEG_BINARY` / `FFMPEG_CPULIMIT` / `FFMPEG_THREADSLIMIT`. If `CPULIMIT` > 0, ffmpeg runs wrapped in the `cpulimit` binary (caps CPU % - vps provider often shutdown VM on CPU peak); `THREADSLIMIT` > 0 adds ffmpeg's `-threads N`. Both are optional — 0 means unlimited.

### Telegram webhook

`telegram-service` is webhook-only (no long-polling mode) — it registers a random per-startup token with Telegram at `TELEGRAM_WEBHOOK_EXTERNALURL/webhook/{token}` (left registered on shutdown so a rolling update does not unregister the new pod's webhook), so that URL must be a real, publicly reachable HTTPS endpoint pointing at the container's `/webhook/{token}` path. Requests to `/webhook/{token}` with an unknown or missing token are rejected with 404, so knowing the public domain alone isn't enough to invoke the handler. Without a reachable URL, the bot never receives updates (this is also why the local compose stack can't fully run Telegram out of the box).

This is unrelated to the OpenRouter/Gemini choice above — the bot just forwards uploads to `storage-service`, which does the actual LLM work regardless of which provider is configured there.

### Local Go development (without Docker)

Start just the infra:
```sh
docker compose up elasticsearch minio postgres -d
```

Then run a service directly, e.g.:
```sh
cd services/storage-service
cp .env.example .env   # edit as needed
go run .
```

**Prerequisite for `storage-service`/`telegram-service`:** libvips must be installed (`vips-dev` / `vips` package).

### Deployment

`deploy/` holds the Ansible playbooks, role templates, and inventories used to deploy to the `test` and `production` environments. The CD pipeline (`.github/workflows/cd.yml`) runs `server_setup.yml` then `deploy.yml` against the target inventory via `workflow_dispatch`.
