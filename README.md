# memelo

A cloud-native meme management system with multi-modal search. Upload images via Telegram, extract text with OCR, generate semantic embeddings, and search your collection by text, fuzzy match, or meaning.

## How to start

In repo root folder:
```bash
cp .env.example .env

# edit .env - must have valid GOOGLE_CREDS_PATH and IMAGE_EMBEDDING_PROJECTNAME

docker compose up -d
```

## How to obtain google creds?
Google docs: https://docs.cloud.google.com/docs/authentication/application-default-credentials  
App need a json file from google cloud console, if you logged in gloud cli - it located at `$HOME/.config/gcloud/application_default_credentials.json`  

So, minimal instruction:
- Register at google cloud
- Enable two apis - `Cloud Vision API` and `Vertex AI API`. (I don't remember where is enable buttons exactly)
- Either:
- - Create service account 
- - Download key for it
- Or: 
- - Install gcloud cli tools
- - Login to your google account. It will create key automatically on your PC in default location.

Anyway, check documentation about ADC, there is a lot of ways to pass credentials to application

## Architecture

```
Integration services  ──(gRPC)─►  storage-service
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

## Local Development

**Start dependencies:**
```sh
 docker compose up elasticsearch minio -d
```

This starts Elasticsearch (`:9200`),  MinIO (`:9000`).

**Build and run storage-service:**
```sh
cd storage-service
cp .env.example .env

# edit .env here
# app must have valid IMAGE_EMBEDDING_PROJECTNAME (google console project id like word-word-111111-a1)

go run .
```

**Prerequisites for storage-service:** libvips must be installed (`vips-dev` / `vips`).

**Docker:**
```sh
docker build -f Dockerfile-storage-service -t memelo-storage .
docker build -f Dockerfile-telegram-service -t memelo-telegram .
```