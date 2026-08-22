data "google_project" "current" {
  project_id = var.project_id
}

module "enable_apis" {
  source = "../modules/enable-apis"

  project_id = var.project_id
  apis = [
    "run.googleapis.com",
    "secretmanager.googleapis.com",
  ]
}

# Reuses the identity of the pre-split shared "runtime-service-account"
# (service_name="runtime", not "ffmpeg") so this directory's resources line
# up with the SA that already exists in state -- avoids a service-account
# swap / Cloud Run revision churn for an already-working deployment.
module "service_account" {
  source = "../modules/service-account"

  project_id   = var.project_id
  environment  = var.environment
  service_name = "runtime"
}
