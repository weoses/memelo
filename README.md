# memelo

A cloud-native meme management system with multi-modal search. Upload images via Telegram, extract text with OCR, generate semantic embeddings, and search your collection by text, fuzzy match, or meaning.

## Architecture

```
Telegram User
     │
     ▼
telegram-service  ──(gRPC)──►  storage-service
                                    │
                          ┌─────────┼─────────┐
                          ▼         ▼         ▼
                     MinIO S3  Elasticsearch  Google Cloud
                    (images)   (metadata +    (Vision OCR +
                               vectors)       Vertex AI embeddings)
```

**Modules:**

| Module | Description |
|---|---|
| `storage-service` | Core service — image processing, OCR, embeddings, search, export |
| `telegram-service` | Telegram bot frontend — upload, search, delete via chat/inline |
| `common` | Shared config, logging, and helper utilities |
| `gen` | Generated protobuf/Connect RPC code (do not edit) |
| `proto` | Protocol buffer source definitions |

## Features

- **Image processing** — converts uploads to JPEG, generates thumbnails (libvips)
- **OCR** — extracts text from images via Google Cloud Vision API
- **Semantic embeddings** — 1408-dimensional vectors via Google Vertex AI (`multimodalembedding@001`)
- **Deduplication** — hash-based and embedding similarity checks on upload to avoid image duplicates
- **Multi-modal search pipeline** — ordered searchers, first match wins:

  | Order | Searcher | Strategy                        |
  |---|---|---------------------------------|
  | 0 | SimpleSearcher | Full-text on OCR result         |
  | 10 | IdSearcher | Direct UUID lookup by image id  |
  | 20 | FuzzySearcher | Fuzzy text match                |
  | 30 | TextEmbeddingSearcher | Semantic vector search          |
  | 40 | AllSearcher | List all (empty query fallback) |


## Tech Stack

- **Language**: Go 1.24
- **API**: Connect RPC (gRPC over HTTP/2)
- **Search**: Elasticsearch 8.16
- **Object storage**: MinIO (S3-compatible)
- **User data**: MongoDB
- **Image processing**: bimg (libvips wrapper) *Require libvips to be installed*
- **OCR**: Google Cloud Vision API
- **Embeddings**: Google Vertex AI
- **DI**: Uber fx
- **Config**: Viper (YAML + env var overrides)

## Configuration

Services are configured via YAML files with environment variable overrides (dots replaced by underscores, e.g. `SERVER_LISTENADDRESS`).

**`storage-service/config.yaml`:**
```yaml
server:
  ListenAddress: :7001

image-storage:
  S3:
    Endpoint: localhost:9000
    Bucket: images

metadata-storage:
  Elastic:
    Addresses: ["http://localhost:9200/"]
  Index: image-metadata
  EmbeddingV1Dimensions: 1408
  EmbeddingMatchTreshold: 0.955

image-embedding:
  ApiEndpoint: us-central1-aiplatform.googleapis.com:443
  ProjectName: <your-gcp-project>
  Model: multimodalembedding@001

image-ocr:
  ApiEndpoint: vision.googleapis.com:443

image-converter:
  ThumbSize: 360
```

**`telegram-service/config.yaml`:**
```yaml
telegram:
  Token: <your-telegram-bot-token>

storage:
  Uri: localhost:7001

mongodb:
  Uri: mongodb://localhost:27017
  Database: memelo

inline:
  PageSize: 10
```

## Local Development

**Start dependencies:**
```sh
cd testenv
docker compose up -d
```

This starts Elasticsearch (`:9200`), Kibana (`:5601`), MinIO (`:9000`), and MongoDB (`:27017`). MinIO auto-creates the `images` bucket.

**Build and run storage-service:**
```sh
cd storage-service
go run .
```

**Build and run telegram-service:**
```sh
cd telegram-service
go run .
```

**Prerequisites for storage-service:** libvips must be installed (`vips-dev` / `vips`).

**Docker:**
```sh
docker build -f Dockerfile-storage-service -t memelo-storage .
docker build -f Dockerfile-telegram-service -t memelo-telegram .
```