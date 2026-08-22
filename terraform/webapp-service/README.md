# webapp-service — Cloud Run (Terraform)

Deploys `webapp-service` to Google Cloud Run using the image published to
`ghcr.io/weoses/webapp-service` by CI.

Publicly reachable (`allow_unauthenticated = true`) — serves the browser
frontend and its JWT-authenticated API to end users. It attaches a
Google-signed ID token on its own outbound calls to storage-service
(`STORAGE_SERVICE_REQUIREGOOGLEIDTOKEN=true`), and grants its own service
account `roles/run.invoker` on it from here (`invoker.tf`).

Not currently included in `docker-compose.yml` (it wasn't before this
change either) — this directory only covers the Cloud Run deployment.

## Prerequisites

- `../global/` and `../storage-service/` must already be applied for the
  target environment (read via `terraform_remote_state`).
- `gcloud auth application-default login`, and `gcloud` authenticated as a
  principal allowed to `run.services.update` (`frontend-url.tf` shells out
  to `gcloud run services update`, same pattern as
  `../telegram-service/webhook.tf` — see that README for why).

## Secrets

`env/<environment>/secrets.enc.env` holds `JWT_SECRET` (sops+PGP). The
committed file has an empty placeholder — fill in a real secret with
`sops env/test/secrets.enc.env` before deploying somewhere real users will
hit (an empty JWT secret means anyone can forge a valid token).

## Usage

```bash
cd terraform/webapp-service
cp terraform.tfvars.example terraform.tfvars
terraform init -backend-config=env/test/backend-test.hcl
terraform plan -var-file=terraform.tfvars
terraform apply -var-file=terraform.tfvars
```
