# Terraform — Cloud Run deployments

Each service has its own directory, its own GCS state (bucket
`melo-terraform-state`, prefix `melo-<environment>/<service>`), and its own
dedicated service account. Shared building blocks live in `modules/`;
shared infrastructure (the two GCS buckets used across services) lives in
`global/`.

```
terraform/
  .sops.yaml            # one PGP key, applies to every env/*/secrets.enc.env below
  modules/
    enable-apis/         # google_project_service, safe to call from multiple states
    service-account/     # one dedicated runtime SA per service
    sops-secret/         # a Secret Manager secret sourced from a sops-encrypted dotenv file
    hmac-key/             # a GCS HMAC key for a service's own SA, stored as two Secret Manager secrets
    bucket-access/        # google_storage_bucket_iam_member, addressed by bucket name (no cross-state coupling)
    cloud-run-service/    # a Cloud Run v2 service + health probes + optional public invoker binding
  global/                # the melo-<env>-temp / melo-<env>-media buckets (lifecycle only, not access grants)
  ffmpeg-service/         # internal, called by storage-service
  youtube-service/        # internal, called by telegram-service
  storage-service/        # internal, called by telegram-service and webapp-service; calls ffmpeg-service
  telegram-service/       # public (Telegram webhook); calls storage-service and youtube-service
  webapp-service/         # public (browser frontend); calls storage-service
```

## Apply order

Dependencies are read cross-state via `terraform_remote_state` (Cloud Run's
default `*.run.app` URL isn't a computable function of project/region/name —
verified live, not just assumed — so it's read from the callee's own already
-applied state rather than guessed). Apply in this order per environment:

1. `global/`
2. `ffmpeg-service/`, `youtube-service/` (either order, no dependency between them)
3. `storage-service/` (needs ffmpeg-service's URI)
4. `telegram-service/`, `webapp-service/` (either order; both need storage-service's URI, telegram-service also needs youtube-service's)

## Inter-service auth

Every internal-only service (ffmpeg, youtube, storage) requires
authenticated Cloud Run invocation — no `allUsers` binding, and each caller
explicitly grants its own service account `roles/run.invoker` on the callee
from the *caller's* own directory (`invoker.tf`), not the callee's. This
keeps "who can call me" and "who am I allowed to call" symmetric and fully
described by whichever side's state you're reading.

On the Go side, `common/auth.GoogleIDTokenInterceptor` attaches a
Google-signed ID token (audienced to the callee's exact URI) to outbound
Connect RPC calls when the corresponding `*_REQUIREGOOGLEIDTOKEN` config
flag is `true` — set only in each caller's Cloud Run `env{}` block here,
never in `docker-compose.yml`. Locally, the flag defaults to `false` and
the interceptor is never constructed — inter-service calls stay exactly as
unauthenticated as they are today.

telegram-service and webapp-service are the two public entry points
(Telegram's webhook, the browser frontend) and allow unauthenticated
invocation themselves, while still attaching ID tokens on their own
outbound calls to the internal services.

## Secrets

Every service's `env/<environment>/secrets.enc.env` is sops-encrypted with
one shared PGP key (`.sops.yaml`, fingerprint
`85A3163A463318C931B03FB80FE2943911B9CCC7`). Edit with `sops env/test/secrets.enc.env`
from inside the relevant service directory — this decrypts to a temp file in
your editor and re-encrypts on save. Committed files ship with the real
secret *keys* the app expects and empty placeholder values; fill in real
values before the corresponding service can do anything beyond boot and
serve `/health`.

## First deploy of a fresh environment

Buckets (`global/`) must be imported if they already exist outside Terraform
(check with `terraform state list` before assuming; `terraform import
google_storage_bucket.temp melo-<env>-temp` / `...media melo-<env>-media`
otherwise create them fresh). Everything else is a plain `terraform apply` in
dependency order above.
