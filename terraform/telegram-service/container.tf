module "container" {
  source = "../modules/cloud-run-service"

  project_id   = var.project_id
  environment  = var.environment
  region       = var.region
  service_name = "telegram-service"
  image        = "ghcr.io/weoses/telegram-service:${var.image_tag}"

  container_port = 7002
  cpu            = var.cpu
  memory         = var.memory
  min_instances  = var.min_instances
  max_instances  = var.max_instances

  service_account_email = module.service_account.email

  env = {
    SERVER_LISTENADDRESS = ":7002"

    # test.memelo.cloud routes to gateway-service, which proxies /webhook
    # through to this service (see ../gateway-service/). Static and known
    # upfront -- no self-referencing-URL bootstrap needed here anymore.
    WEBHOOK_EXTERNALURL = "https://${var.domain_name}/webhook"

    STORAGE_SERVICE_URI                  = data.terraform_remote_state.storage.outputs.uri
    STORAGE_SERVICE_REQUIREGOOGLEIDTOKEN = "true"

    YOUTUBE_SERVICE_URI                  = data.terraform_remote_state.youtube.outputs.uri
    YOUTUBE_SERVICE_REQUIREGOOGLEIDTOKEN = "true"

    TEMP_STORAGE_ENDPOINT = "storage.googleapis.com"
    TEMP_STORAGE_BUCKET   = "melo-${var.environment}-temp"
    TEMP_STORAGE_SECURE   = "true"

    LOG_FORMAT    = "json"
    LOG_PROJECTID = var.project_id
  }

  secret_env = {
    TEMP_STORAGE_ACCESSKEY = module.hmac_key.access_key_secret_id
    TEMP_STORAGE_SECRETKEY = module.hmac_key.secret_key_secret_id
  }

  secrets_volume_secret_id = module.secrets.secret_id

  # telegram-service is purely internal now: gateway-service is the one
  # public entry point and proxies /webhook here, attaching its own Google
  # ID token (invoker grant lives in ../gateway-service/invoker.tf).
  allow_unauthenticated = false
}
