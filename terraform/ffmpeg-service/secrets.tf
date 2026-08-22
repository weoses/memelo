# ffmpeg-service has no real secrets of its own today -- this placeholder
# keeps a non-empty file mounted at /etc/secrets/secrets so the shared
# scripts/entrypoint.sh (which sources every file matching
# /etc/secrets/*/*) always has at least one real file to find. Without a
# fix landed in a newer image, an empty glob there makes entrypoint.sh abort
# under `set -e` before the app ever starts. Drop this file once a rebuilt
# image containing the entrypoint.sh fix is deployed.
module "secrets" {
  source = "../modules/sops-secret"

  project_id      = var.project_id
  secret_id       = "${var.environment}-ffmpeg-secrets"
  source_file     = "env/${var.environment}/secrets.enc.env"
  accessor_member = module.service_account.member
}
