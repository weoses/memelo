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
# edit terraform.tfvars: project_id, image, etc.

terraform init -backend-config=backend-test.hcl   # or backend-prod.hcl
terraform plan -var-file=terraform.tfvars
terraform apply -var-file=terraform.tfvars
```

Switching between environments (e.g. test -> prod) requires re-running
`terraform init -backend-config=backend-<env>.hcl -reconfigure` to point at the
other bucket.

## Notes

- `config.yaml` is no longer templated or served from GCS. The image's
  Dockerfile (`Dockerfile-ffmpeg-service`) bakes `ffmpeg-service/config.yaml`
  (the same default used for local runs) into `/app/config/config.yaml`,
  matching `APPLICATION_CONFIGPATH=/app/config` — so Viper picks it up the
  same way it does locally, no GCS volume or bucket involved. Values that
  need to differ from that default in Cloud Run (`server.ListenAddress`,
  `temp-storage.Endpoint`/`Bucket`/`Secure`, `log.Level`) are set as plain
  container `env {}` vars in `main.tf` (`SERVER_LISTENADDRESS`,
  `TEMP_STORAGE_ENDPOINT`, `TEMP_STORAGE_BUCKET`, `TEMP_STORAGE_SECURE`,
  `LOG_LEVEL`) — Viper's `AutomaticEnv` overrides file values with these, same
  mechanism the mounted secret files rely on. `SERVER_LISTENADDRESS` is set to
  `:<container_port>` (default `:8080`) so the app binds to
  `0.0.0.0:$container_port` as Cloud Run requires — the app's own default of
  `localhost:7005` (from `config.yaml`) would not work on Cloud Run.
- Real credentials go through `secret_names` instead: a list of Secret Manager
  secret IDs, e.g. `["temp-storage-secrets"]`. Terraform grants access
  (`roles/secretmanager.secretAccessor` on each secret) and mounts the
  `latest` version of each into the container under `/etc/secrets/<name>/<name>`.
  If a name in `secret_names` doesn't already exist as a secret by the time
  `apply` runs, apply fails (the IAM binding / volume reference a nonexistent
  resource).
  - `temp-storage-secrets` is managed end-to-end by Terraform via
    [`secrets.tf`](secrets.tf): its content comes from the
    [sops](https://github.com/getsops/sops)-encrypted
    `env/<environment>/temp-storage.enc.env`, decrypted locally by the
    `carlpett/sops` provider at plan/apply time and written straight into a
    new Secret Manager secret version. Only ciphertext ever touches disk or
    git — plaintext only exists in Terraform's in-memory plan and in the
    resulting Secret Manager version (and, as usual for any secret value
    Terraform manages, in the remote state file).
    - Requires a `.sops.yaml` (or equivalent inline sops config) with a key
      (GCP KMS, age, etc.) that whoever runs `apply` has access to decrypt
      with. This repo doesn't ship one yet — add it before encrypting real
      values.
    - To create/update the encrypted file:
      ```bash
      # first time, from a plaintext KEY=VALUE draft:
      sops --input-type dotenv --output-type dotenv \
        -e temp-storage.env > env/test/secrets.enc.env

      # edit an existing encrypted file in place:
      sops env/test/secrets.enc.env
      ```
      Then re-run `terraform apply` to push the new content as a fresh
      secret version and roll out a new Cloud Run revision.
    - Any *other* names in `secret_names` besides `temp-storage-secrets` are
      still expected to be created and populated **outside Terraform**
      (`gcloud secrets create <name> --data-file=-`) — Terraform only
      creates/populates the one backed by `secrets.tf`; for everything else
      it never creates a secret, writes a version, or reads/parses content.

  The image's entrypoint script (`ffmpeg-service/entrypoint.sh`) sources
  every mounted file before exec'ing the app, so the app still just sees
  plain env vars via Viper's `AutomaticEnv`, same as today. Cloud Run can
  only bind a whole secret to a single discrete env var, so multi-key files
  like these are necessarily mounted as files rather than as native Cloud
  Run "secret env vars".

  To rotate `temp-storage-secrets`, edit it via `sops` and re-`apply` (see
  above). For any externally-managed secret, add a new version out-of-band
  (`gcloud secrets versions add <name> --data-file=-`) and deploy a new
  Cloud Run revision (e.g. re-`apply`) to pick up `latest`.
- `ingress` is set to `INGRESS_TRAFFIC_ALL` with access controlled purely by IAM
  (`invoker_members`). If all callers are on the same VPC (e.g. via Serverless VPC
  Access / a GCP-hosted storage-service), consider tightening this to
  `INGRESS_TRAFFIC_INTERNAL_ONLY`.
- No Artifact Registry is created here; if/when other services get Terraform
  configs, consider moving the image build/push to GCP Artifact Registry instead of
  ghcr.io.
