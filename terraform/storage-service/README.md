# storage-service — Cloud Run (Terraform)

Deploys `storage-service` to Google Cloud Run using the image published to
`ghcr.io/weoses/storage-service` by CI.

Authenticated-only (no `allUsers` invoker binding) — called by
telegram-service and webapp-service, both of which grant their own service
account `roles/run.invoker` on this service from their own directories.
storage-service itself calls ffmpeg-service, and grants its own service
account `roles/run.invoker` on it here (`invoker.tf`), attaching a
Google-signed ID token via `common/auth.GoogleIDTokenInterceptor` on the Go
side (`FFMPEG_SERVICE_REQUIREGOOGLEIDTOKEN=true`).

## Prerequisites

- `../global/` and `../ffmpeg-service/` must already be applied for the
  target environment — this config reads ffmpeg-service's Cloud Run URI via
  `terraform_remote_state` (`invoker.tf`), so ffmpeg-service must exist
  first.
- `gcloud auth application-default login`.

## Secrets

`env/<environment>/secrets.enc.env` (sops+PGP, same key as every other
service) holds, as a multi-key dotenv file mounted at `/etc/secrets/secrets`
and sourced by the image's `entrypoint.sh`:

- `METADATA_DB_ELASTIC_CLOUDID`, `METADATA_DB_ELASTIC_APIKEY`, `METADATA_DB_INDEX`
- `TAG_DB_ELASTIC_CLOUDID`, `TAG_DB_ELASTIC_APIKEY`, `TAG_DB_INDEX`
- `GEMINI_EXTRACTOR_APIKEY`, `GEMINI_EMBEDDING_APIKEY`
- `OPENROUTER_EXTRACTOR_APIKEY`, `OPENROUTER_EMBEDDING_APIKEY`

The committed file has these keys with empty placeholder values — fill in
real values with `sops env/test/secrets.enc.env` (opens your PGP-encrypted
editor session) before the service can actually reach Elastic/Gemini/OpenRouter.
The container will deploy and pass `/health` even with empty values (nothing
reads them at startup), but pipeline/search calls that hit Elastic or the LLM
providers will fail until real values are set — re-run `terraform apply`
after editing to push a new secret version and roll a new revision.

No `GOOGLE_APPLICATION_CREDENTIALS` / keyfile is configured in Cloud Run:
Cloud Run's metadata server already provides Application Default Credentials
for this service's own service account, and populating a Gemini API key
routes the genai client through the API-key backend (no ADC needed at all).
That keyfile mount stays a docker-compose-only, local-dev mechanism.

## Usage

```bash
cd terraform/storage-service
cp terraform.tfvars.example terraform.tfvars
terraform init -backend-config=env/test/backend-test.hcl
terraform plan -var-file=terraform.tfvars
terraform apply -var-file=terraform.tfvars
```
