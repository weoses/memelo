# Keys only -- values are supplied by whoever owns them (Elastic Cloud,
# Gemini/OpenRouter API keys) via `sops env/<environment>/secrets.enc.env`,
# not committed here.
module "secrets" {
  source = "../modules/sops-secret"

  project_id      = var.project_id
  secret_id       = "${var.environment}-storage-secrets"
  source_file     = "env/${var.environment}/secrets.enc.env"
  accessor_member = module.service_account.member
}
