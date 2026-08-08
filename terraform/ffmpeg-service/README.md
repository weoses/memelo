# ffmpeg-service — Cloud Run (Terraform)

Deploys `ffmpeg-service` to Google Cloud Run using the existing image published to
`ghcr.io/weoses/ffmpeg-service` by CI. No GCP Artifact Registry is provisioned —
Cloud Run pulls the public ghcr.io image directly.

The service is deployed **authenticated-only**: no `allUsers` invoker binding is
created. Grant access to specific callers (e.g. storage-service's runtime service
account) via `invoker_members`.

## Prerequisites

- `gcloud auth application-default login` (or a service account key) with
  permissions to manage Cloud Run and enable APIs on the target project.
- A GCP project with billing enabled. The Cloud Run API
  (`run.googleapis.com`) is enabled automatically by this config.
- The state bucket for the environment you're targeting must already exist
  (`melo-test-terraform` / `melo-prod-terraform`) — see "State backend" below.

## State backend

State is stored remotely in GCS instead of a local `terraform.tfstate` file, one
bucket per environment:

- test: `gs://melo-test-terraform`
- prod: `gs://melo-prod-terraform`

Both live in the same GCP project as the rest of this config. The bucket name is
intentionally left out of `versions.tf` (partial backend configuration) since it
differs per environment — pass it via `-backend-config` at init time using the
provided `backend-test.hcl` / `backend-prod.hcl` files. State objects are stored
under the `ffmpeg-service` prefix so the bucket can be shared with other services'
Terraform configs later.

## Usage

```bash
cd terraform/ffmpeg-service
cp terraform.tfvars.example terraform.tfvars
# edit terraform.tfvars: project_id, temp_storage_* values, etc.

terraform init -backend-config=backend-test.hcl   # or backend-prod.hcl
terraform plan -var-file=terraform.tfvars
terraform apply -var-file=terraform.tfvars
```

Switching between environments (e.g. test -> prod) requires re-running
`terraform init -backend-config=backend-<env>.hcl -reconfigure` to point at the
other bucket.

## Notes

- `config.yaml` (rendered from `templates/config.yaml.tftpl`) is uploaded as a
  plain object to a dedicated GCS bucket and mounted read-only at `/app/config`
  via a Cloud Run GCS volume — matching `APPLICATION_CONFIGPATH=/app/config`
  baked into the image, so Viper picks it up the same way it does locally.
  All non-secret settings live in this file, including `server.ListenAddress`
  (set to `:<container_port>`, default `:8080`, so the app binds to
  `0.0.0.0:$container_port` as Cloud Run requires — the app's own default of
  `localhost:7005` would not work on Cloud Run) and `temp-storage.Endpoint`/
  `Bucket`/`Secure`. Nothing is passed as plain container env vars anymore; the
  container has no `env {}` blocks at all. The Cloud Run runtime service account
  (`service_account_email`, or the project's default compute service account if
  unset) is granted `roles/storage.objectViewer` on the bucket automatically.
- Real credentials go through `secret_names` instead: a list of Secret Manager
  secret IDs, e.g. `["temp-storage-secrets"]`. These secrets are created and
  populated **outside Terraform** — typically:
  ```bash
  printf 'TEMP_STORAGE_ACCESSKEY=...\nTEMP_STORAGE_SECRETKEY=...\n' | \
    gcloud secrets create temp-storage-secrets --data-file=- --project=<project-id>
  ```
  Terraform only grants access (`roles/secretmanager.secretAccessor` on each
  secret) and mounts the `latest` version of each into the container under
  `/etc/secrets/<name>/<name>` — it never creates a secret, writes a version, or
  reads/parses the content. If a name in `secret_names` doesn't already exist as
  a secret, `apply` fails outright (the IAM binding / volume reference a
  nonexistent resource). The image's entrypoint script
  (`ffmpeg-service/entrypoint.sh`) sources every mounted file before exec'ing the
  app, so the app still just sees plain env vars via Viper's `AutomaticEnv`, same
  as today. Cloud Run can only bind a whole secret to a single discrete env var,
  so multi-key files like these are necessarily mounted as files rather than as
  native Cloud Run "secret env vars".
  To rotate a credential, add a new secret version out-of-band
  (`gcloud secrets versions add temp-storage-secrets --data-file=-`) and deploy a
  new Cloud Run revision (e.g. re-`apply`) to pick up `latest`.
- `ingress` is set to `INGRESS_TRAFFIC_ALL` with access controlled purely by IAM
  (`invoker_members`). If all callers are on the same VPC (e.g. via Serverless VPC
  Access / a GCP-hosted storage-service), consider tightening this to
  `INGRESS_TRAFFIC_INTERNAL_ONLY`.
- No Artifact Registry is created here; if/when other services get Terraform
  configs, consider moving the image build/push to GCP Artifact Registry instead of
  ghcr.io.
