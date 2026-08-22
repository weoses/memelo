module "hmac_key" {
  source = "../modules/hmac-key"

  project_id            = var.project_id
  environment           = var.environment
  service_name          = "ffmpeg"
  service_account_email = module.service_account.email
  accessor_member       = module.service_account.member
}
