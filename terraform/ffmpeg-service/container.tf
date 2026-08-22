module "container" {
  source = "../modules/cloud-run-service"

  project_id   = var.project_id
  environment  = var.environment
  region       = var.region
  service_name = "ffmpeg-service"
  image        = "ghcr.io/weoses/ffmpeg-service:${var.image_tag}"

  container_port = 7003
  cpu            = var.cpu
  memory         = var.memory
  min_instances  = var.min_instances
  max_instances  = var.max_instances

  service_account_email = module.service_account.email

  env = {
    SERVER_LISTENADDRESS  = ":7003"
    TEMP_STORAGE_ENDPOINT = var.google_storage_host
    TEMP_STORAGE_BUCKET   = "melo-${var.environment}-temp"
    TEMP_STORAGE_SECURE   = "true"
  }

  secret_env = {
    TEMP_STORAGE_ACCESSKEY = module.hmac_key.access_key_secret_id
    TEMP_STORAGE_SECRETKEY = module.hmac_key.secret_key_secret_id
  }

  # ffmpeg-service is purely internal: only storage-service calls it, and
  # that grant lives in storage-service's own directory (invoker.tf there),
  # not here. No allUsers binding.
  allow_unauthenticated = false
}
