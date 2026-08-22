module "secrets" {
  source = "../modules/sops-secret"

  project_id      = var.project_id
  secret_id       = "${var.environment}-telegram-secrets"
  source_file     = "env/${var.environment}/secrets.enc.env"
  accessor_member = module.service_account.member
}
