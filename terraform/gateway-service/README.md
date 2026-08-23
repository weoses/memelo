# gateway-service — Cloud Run (Terraform)

Deploys `gateway-service` to Google Cloud Run using the existing image
published to `ghcr.io/weoses/gateway-service` by CI.

The service is deployed **publicly** (`allow_unauthenticated = true`) — it's
the one intentional public entry point in this stack. It reverse-proxies
`/webhook` to telegram-service and everything else (behind Basic Auth) to
webapp-service, both of which are IAM-protected and reachable only via this
service's own invoker grants (`invoker.tf`).

## Prerequisites

Same as `../ffmpeg-service/README.md` — `gcloud auth application-default
login`, and `../global/` already applied for the target environment. Also
requires `../telegram-service/` and `../webapp-service/` already applied
(their `.outputs.uri`/`.outputs.name` are read via `terraform_remote_state`).

## State backend

GCS bucket `melo-terraform-state`, prefix `melo-<environment>/gateway-service`.

## Usage

```bash
cd terraform/gateway-service
cp terraform.tfvars.example terraform.tfvars
terraform init -backend-config=env/test/backend-test.hcl
terraform plan -var-file=terraform.tfvars
terraform apply -var-file=terraform.tfvars
```

## Notes

- `env/test/secrets.enc.env` holds `BASIC_AUTH_USERNAME` / `BASIC_AUTH_PASSWORD`,
  sops+PGP encrypted (same key as every other service, `../.sops.yaml`).
- `domain.tf`'s `google_cloud_run_domain_mapping` resource was created
  outside Terraform (via `gcloud`/console) before this directory existed and
  originally routed directly at `test-webapp-service`. It must be imported
  (`terraform import google_cloud_run_domain_mapping.this <import-id>`,
  confirm the exact ID format against the google provider docs) before the
  first `apply` here, which then re-points it at this service.
- After this service exists, `../telegram-service/container.tf` and
  `../webapp-service/container.tf` set their externally-visible URL env vars
  (`WEBHOOK_EXTERNALURL`, `FRONTEND_BASEURL`) to `https://${var.domain_name}`
  directly — a plain declarative value, not a remote-state read from here,
  since the domain is static and known upfront.
