module "container" {
  source = "../modules/cloud-run-service"

  project_id   = var.project_id
  environment  = var.environment
  region       = var.region
  service_name = "youtube-service"
  image        = "ghcr.io/weoses/youtube-service:${var.image_tag}"

  container_port = 7004
  cpu            = var.cpu
  memory         = var.memory
  min_instances  = var.min_instances
  max_instances  = var.max_instances

  service_account_email = module.service_account.email

  env = {
    SERVER_LISTENADDRESS           = ":7004"
    TEMP_STORAGE_ENDPOINT          = "storage.googleapis.com"
    TEMP_STORAGE_BUCKET            = "melo-${var.environment}-temp"
    TEMP_STORAGE_SECURE            = "true"
    YOUTUBE_MAXVIDEOSIZEBYTES      = tostring(var.max_video_size_bytes)
    YOUTUBE_MAXCONCURRENTDOWNLOADS = tostring(var.max_concurrent_downloads)

    LOG_FORMAT    = "json"
    LOG_PROJECTID = var.project_id
  }

  secret_env = {
    TEMP_STORAGE_ACCESSKEY = module.hmac_key.access_key_secret_id
    TEMP_STORAGE_SECRETKEY = module.hmac_key.secret_key_secret_id
  }

  secrets_volume_secret_id = module.secrets.secret_id

  # youtube-service is purely internal: only telegram-service calls it, and
  # that grant lives in telegram-service's own directory (invoker.tf there).
  allow_unauthenticated = false
}
