module "container" {
  source = "../modules/cloud-run-service"

  project_id   = var.project_id
  environment  = var.environment
  region       = var.region
  service_name = "webapp-service"
  image        = "ghcr.io/weoses/webapp-service:${var.image_tag}"

  container_port = 7006
  cpu            = var.cpu
  memory         = var.memory
  min_instances  = var.min_instances
  max_instances  = var.max_instances

  service_account_email = module.service_account.email

  env = {
    SERVER_LISTENADDRESS = ":7006"

    STORAGE_SERVICE_URI                  = data.terraform_remote_state.storage.outputs.uri
    STORAGE_SERVICE_REQUIREGOOGLEIDTOKEN = "true"

    TEMP_STORAGE_ENDPOINT = "storage.googleapis.com"
    TEMP_STORAGE_BUCKET   = "melo-${var.environment}-temp"
    TEMP_STORAGE_SECURE   = "true"

    # Real value patched in by frontend-url.tf right after this resource
    # applies -- same self-referencing-URL problem as telegram-service's
    # webhook, see that service's README for the full explanation. Not
    # startup-critical here (read per-request from already-served HTML,
    # not at boot), but still worth getting right automatically.
    FRONTEND_BASEURL = ""
  }

  secret_env = {
    TEMP_STORAGE_ACCESSKEY = module.hmac_key.access_key_secret_id
    TEMP_STORAGE_SECRETKEY = module.hmac_key.secret_key_secret_id
  }

  secrets_volume_secret_id = module.secrets.secret_id

  # webapp-service serves the public browser frontend -- unauthenticated
  # invocation required.
  allow_unauthenticated = true
}
