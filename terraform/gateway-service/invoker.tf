data "terraform_remote_state" "telegram" {
  backend = "gcs"
  config = {
    bucket = "melo-terraform-state"
    prefix = "melo-${var.environment}/telegram-service"
  }
}

data "terraform_remote_state" "webapp" {
  backend = "gcs"
  config = {
    bucket = "melo-terraform-state"
    prefix = "melo-${var.environment}/webapp-service"
  }
}

resource "google_cloud_run_v2_service_iam_member" "telegram_invoker" {
  project  = var.project_id
  location = var.region
  name     = data.terraform_remote_state.telegram.outputs.name
  role     = "roles/run.invoker"
  member   = module.service_account.member
}

resource "google_cloud_run_v2_service_iam_member" "webapp_invoker" {
  project  = var.project_id
  location = var.region
  name     = data.terraform_remote_state.webapp.outputs.name
  role     = "roles/run.invoker"
  member   = module.service_account.member
}
