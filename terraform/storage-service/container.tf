module "container" {
  source = "../modules/cloud-run-service"

  project_id   = var.project_id
  environment  = var.environment
  region       = var.region
  service_name = "storage-service"
  image        = "ghcr.io/weoses/storage-service:${var.image_tag}"

  container_port = 7001
  cpu            = var.cpu
  memory         = var.memory
  min_instances  = var.min_instances
  max_instances  = var.max_instances

  service_account_email = module.service_account.email

  env = {
    SERVER_LISTENADDRESS = ":7001"

    MEDIA_STORAGE_ENDPOINT = "storage.googleapis.com"
    MEDIA_STORAGE_BUCKET   = "melo-${var.environment}-media"
    MEDIA_STORAGE_SECURE   = "true"

    TEMP_STORAGE_ENDPOINT = "storage.googleapis.com"
    TEMP_STORAGE_BUCKET   = "melo-${var.environment}-temp"
    TEMP_STORAGE_SECURE   = "true"

    MEDIA_EMBEDDING_PROJECTNAME = var.embedding_project_name
    AUDIO_STT_PROJECTNAME       = var.embedding_project_name

    EXTRACTOR_PROVIDER = var.extractor_provider
    EMBEDDER_PROVIDER  = var.embedder_provider

    FFMPEG_SERVICE_URI                  = data.terraform_remote_state.ffmpeg.outputs.uri
    FFMPEG_SERVICE_REQUIREGOOGLEIDTOKEN = "true"

    LOG_FORMAT    = "json"
    LOG_PROJECTID = var.project_id
  }

  secret_env = {
    MEDIA_STORAGE_ACCESSKEY = module.hmac_key.access_key_secret_id
    MEDIA_STORAGE_SECRETKEY = module.hmac_key.secret_key_secret_id
    TEMP_STORAGE_ACCESSKEY  = module.hmac_key.access_key_secret_id
    TEMP_STORAGE_SECRETKEY  = module.hmac_key.secret_key_secret_id
  }

  secrets_volume_secret_id = module.secrets.secret_id

  # storage-service is purely internal: called by telegram-service and
  # webapp-service, both of which declare their own invoker grants.
  allow_unauthenticated = false
}
