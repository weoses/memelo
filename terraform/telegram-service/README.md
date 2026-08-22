# telegram-service — Cloud Run (Terraform)

Deploys `telegram-service` to Google Cloud Run using the image published to
`ghcr.io/weoses/telegram-service` by CI.

Publicly reachable (`allow_unauthenticated = true`) — Telegram's servers POST
webhook updates to this service directly, so it can't require Google-signed
caller auth. It still attaches a Google-signed ID token on its own outbound
calls to storage-service and youtube-service
(`STORAGE_SERVICE_REQUIREGOOGLEIDTOKEN` / `YOUTUBE_SERVICE_REQUIREGOOGLEIDTOKEN`),
and grants its own service account `roles/run.invoker` on both from here
(`invoker.tf`).

## Prerequisites

- `../global/`, `../storage-service/`, and `../youtube-service/` must already
  be applied for the target environment (read via `terraform_remote_state`
  in `invoker.tf`/`container.tf`).
- `gcloud auth application-default login`, and `gcloud` itself installed and
  authenticated as a principal allowed to `run.services.update` — `webhook.tf`
  shells out to `gcloud run services update` as part of every apply (see
  below).

## The webhook URL problem

Cloud Run's default `*.run.app` URL isn't known until after the service is
first created, but `telegram-service` needs to know its **own** public URL to
register a webhook with Telegram at startup
(`telegram-service/service/Telegram.go`, `RegisterWebhook`) — and it does
this unconditionally, so an invalid URL there blocks the app from ever
starting.

This is solved in two parts:
- `container.tf` sets `WEBHOOK_EXTERNALURL` to a syntactically-valid but
  fake placeholder so the very first deploy can boot and register *something*
  with Telegram.
- `webhook.tf` runs `gcloud run services update ... --update-env-vars=WEBHOOK_EXTERNALURL=<real uri>/webhook`
  as a `local-exec` provisioner right after the Cloud Run resource applies,
  using the now-known real URI. It's not possible to reference a resource's
  own computed URI from within its own config (a genuine cycle), so this has
  to be an imperative follow-up step, not a declarative one. It runs on
  *every* `terraform apply` (not just the first), because Terraform's own
  `env` list doesn't declare the real value and would otherwise reset it
  back to the placeholder on the next unrelated change.

## Secrets

`env/<environment>/secrets.enc.env` holds `TELEGRAM_TOKEN` (sops+PGP). The
committed file has an empty placeholder — fill in the real bot token with
`sops env/test/secrets.enc.env` before the bot can actually talk to Telegram.

## Usage

```bash
cd terraform/telegram-service
cp terraform.tfvars.example terraform.tfvars
terraform init -backend-config=env/test/backend-test.hcl
terraform plan -var-file=terraform.tfvars
terraform apply -var-file=terraform.tfvars
```
