# WEBHOOK_EXTERNALURL is deliberately NOT set here -- it needs this
# service's own Cloud Run URL, which isn't known until after the service is
# first created (Cloud Run's default *.run.app URL isn't a computable
# function of project/region/name). See webhook.tf, which patches it in via
# `gcloud run services update` once the URL is known, and re-patches it on
# every apply since Terraform's own config here doesn't declare it (and
# would otherwise strip it back out on the next apply).
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

    # Real value patched in by webhook.tf right after this resource applies
    # (see that file for why). This placeholder only matters on the very
    # first-ever creation, so the app has *some* URL to register with
    # Telegram and pass its own startup health check -- Telegram's
    # setWebhook actually resolves DNS and connects synchronously (a
    # made-up host like "placeholder.invalid" gets rejected outright with
    # "Failed to resolve host"), so this has to be a real, reachable HTTPS
    # endpoint. storage-service's own already-live URL is a safe, hermetic
    # choice (no external third-party dependency) -- Telegram will happily
    # connect to it and get some 403/404, which is enough to accept the
    # registration.
    WEBHOOK_EXTERNALURL = "${data.terraform_remote_state.storage.outputs.uri}/webhook"

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

  # telegram-service needs a public webhook endpoint reachable by Telegram's
  # servers -- unlike storage/ffmpeg/youtube, it allows unauthenticated
  # invocation.
  allow_unauthenticated = true
}
