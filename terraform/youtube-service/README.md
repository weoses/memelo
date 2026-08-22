# youtube-service — Cloud Run (Terraform)

Deploys `youtube-service` to Google Cloud Run using the existing image published
to `ghcr.io/weoses/youtube-service` by CI.

The service is deployed **authenticated-only**: no `allUsers` invoker binding.
youtube-service is purely internal (only telegram-service calls it); the
invoker grant for telegram-service's own service account lives in
`../telegram-service/invoker.tf`, not here.

## Prerequisites

Same as `../ffmpeg-service/README.md` — `gcloud auth application-default
login`, and `../global/` already applied for the target environment.

## State backend

GCS bucket `melo-terraform-state`, prefix `melo-<environment>/youtube-service`.

## Usage

```bash
cd terraform/youtube-service
cp terraform.tfvars.example terraform.tfvars
terraform init -backend-config=env/test/backend-test.hcl
terraform plan -var-file=terraform.tfvars
terraform apply -var-file=terraform.tfvars
```

## Notes

- `env/test/secrets.enc.env` holds `YOUTUBE_APIKEY`, sops+PGP encrypted (same
  key as every other service, `../.sops.yaml`).
- This directory was split out of what used to be a single combined
  `terraform/ffmpeg-service/` config that deployed both ffmpeg-service and
  youtube-service. youtube-service now gets its own dedicated service
  account (rather than continuing to share ffmpeg-service's), which means
  its very first apply after the split rolls one new Cloud Run revision.
