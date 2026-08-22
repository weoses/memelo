# Cloud Run's default *.run.app URL is NOT a deterministic function of
# project number/region (verified live: two services in the same
# project+region shared one opaque hash, but that hash isn't something HCL
# can compute) -- so cross-service URIs are read via terraform_remote_state
# from the callee's own already-applied state, not guessed as a local.
data "terraform_remote_state" "ffmpeg" {
  backend = "gcs"
  config = {
    bucket = "melo-terraform-state"
    prefix = "melo-${var.environment}/ffmpeg-service"
  }
}

resource "google_cloud_run_v2_service_iam_member" "ffmpeg_invoker" {
  project  = var.project_id
  location = var.region
  name     = data.terraform_remote_state.ffmpeg.outputs.name
  role     = "roles/run.invoker"
  member   = module.service_account.member
}
