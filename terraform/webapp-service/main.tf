module "enable_apis" {
  source = "../modules/enable-apis"

  project_id = var.project_id
  apis = [
    "run.googleapis.com",
    "secretmanager.googleapis.com",
  ]
}

module "service_account" {
  source = "../modules/service-account"

  project_id   = var.project_id
  environment  = var.environment
  service_name = "webapp"
}
