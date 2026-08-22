data "terraform_remote_state" "storage" {
  backend = "gcs"
  config = {
    bucket = "melo-terraform-state"
    prefix = "melo-${var.environment}/storage-service"
  }
}

data "terraform_remote_state" "youtube" {
  backend = "gcs"
  config = {
    bucket = "melo-terraform-state"
    prefix = "melo-${var.environment}/youtube-service"
  }
}

resource "google_cloud_run_v2_service_iam_member" "storage_invoker" {
  project  = var.project_id
  location = var.region
  name     = data.terraform_remote_state.storage.outputs.name
  role     = "roles/run.invoker"
  member   = module.service_account.member
}

resource "google_cloud_run_v2_service_iam_member" "youtube_invoker" {
  project  = var.project_id
  location = var.region
  name     = data.terraform_remote_state.youtube.outputs.name
  role     = "roles/run.invoker"
  member   = module.service_account.member
}
