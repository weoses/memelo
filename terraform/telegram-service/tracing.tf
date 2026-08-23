# See ../ffmpeg-service/tracing.tf for why this grant is required (not
# covered by Cloud Logging's free ride).
resource "google_project_iam_member" "cloudtrace_agent" {
  project = var.project_id
  role    = "roles/cloudtrace.agent"
  member  = module.service_account.member
}
