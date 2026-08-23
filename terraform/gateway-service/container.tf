module "container" {
  source = "../modules/cloud-run-service"

  project_id   = var.project_id
  environment  = var.environment
  region       = var.region
  service_name = "gateway-service"
  image        = "ghcr.io/weoses/gateway-service:${var.image_tag}"

  container_port = 7008
  cpu            = var.cpu
  memory         = var.memory
  min_instances  = var.min_instances
  max_instances  = var.max_instances

  service_account_email = module.service_account.email

  env = {
    SERVER_LISTENADDRESS = ":7008"

    TELEGRAM_SERVICE_URI                  = data.terraform_remote_state.telegram.outputs.uri
    TELEGRAM_SERVICE_REQUIREGOOGLEIDTOKEN = "true"
    WEBAPP_SERVICE_URI                    = data.terraform_remote_state.webapp.outputs.uri
    WEBAPP_SERVICE_REQUIREGOOGLEIDTOKEN   = "true"

    LOG_FORMAT    = "json"
    LOG_PROJECTID = var.project_id
  }

  secrets_volume_secret_id = module.secrets.secret_id

  # gateway-service is the one intentionally public entry point: it proxies
  # /webhook to telegram-service and everything else (behind Basic Auth) to
  # webapp-service, both of which are IAM-protected and reachable only via
  # this service's own invoker grants (invoker.tf).
  allow_unauthenticated = true
}
