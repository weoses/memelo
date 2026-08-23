# Cloud Logging works today with zero IAM -- Cloud Run captures container
# stdout independently of the runtime SA. That free ride does NOT cover the
# Cloud Trace *write* API, which the app's OTel exporter calls using the
# runtime SA's ADC -- without this grant, spans are silently dropped
# (non-fatal, but Cloud Trace stays empty with no error surfaced anywhere).
resource "google_project_iam_member" "cloudtrace_agent" {
  project = var.project_id
  role    = "roles/cloudtrace.agent"
  member  = module.service_account.member
}
