# ffmpeg-service — Cloud Run (Terraform)

Deploys `ffmpeg-service` to Google Cloud Run using the existing image published to
`ghcr.io/weoses/ffmpeg-service` by CI. No GCP Artifact Registry is provisioned —
Cloud Run pulls the public ghcr.io image directly.

The service is deployed **authenticated-only**: no `allUsers` invoker binding.
ffmpeg-service is purely internal (only storage-service calls it); the invoker
grant for storage-service's own service account lives in
`../storage-service/invoker.tf`, not here — see `../README.md` for how the
modular layout and cross-service IAM grants fit together.

## Prerequisites

- `gcloud auth application-default login` with permissions to manage Cloud
  Run, Secret Manager, and IAM on the target project.
- The shared state bucket `melo-terraform-state` and the buckets managed by
  `../global/` must already exist for the environment you're targeting.

## State backend

State is stored remotely in GCS: bucket `melo-terraform-state`, prefix
`melo-<environment>/ffmpeg-service`. Pass the environment's backend file via
`-backend-config` at init time.

## Usage

```bash
cd terraform/ffmpeg-service
cp terraform.tfvars.example terraform.tfvars
# edit terraform.tfvars: project_id, image_tag, etc.

terraform init -backend-config=env/test/backend-test.hcl   # or env/prod/backend-prod.hcl
terraform plan -var-file=terraform.tfvars
terraform apply -var-file=terraform.tfvars
```

Switching between environments (e.g. test -> prod) requires re-running
`terraform init -backend-config=env/<env>/backend-<env>.hcl -reconfigure`.

## Notes

- `config.yaml` is baked into the image (`Dockerfile-ffmpeg-service`), same as
  local runs. Values that need to differ in Cloud Run
  (`server.ListenAddress`, `temp-storage.*`) are set as plain container
  `env {}` vars in `container.tf` — Viper's `AutomaticEnv` overrides file
  values with these. `SERVER_LISTENADDRESS` is `:7003` so the app binds
  `0.0.0.0:7003` as Cloud Run requires (the app's own `localhost:7003`
  default from `config.yaml` would not work on Cloud Run).
- Temp storage credentials (`TEMP_STORAGE_ACCESSKEY`/`SECRETKEY`) come from a
  GCS HMAC key created for this service's own dedicated service account
  (`../modules/hmac-key`), mounted as native Cloud Run secret-backed env
  vars.
- `secrets.tf` mounts a placeholder secret (`env/<environment>/secrets.enc.env`,
  currently just `_PLACEHOLDER=1`) — ffmpeg-service has no real secrets of its
  own, but the image's `entrypoint.sh` needs at least one file under
  `/etc/secrets/*/*` to exist or it aborts (fixed in source, but requires a
  rebuilt image to take effect — see `scripts/entrypoint.sh`). Drop this file
  and `secrets.tf` once a rebuilt image is deployed.
