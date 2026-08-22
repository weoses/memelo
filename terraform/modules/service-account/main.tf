# One dedicated runtime service account per deployed Cloud Run service.
#
# service_name must be a short slug (e.g. "storage", "ffmpeg"), NOT the full
# "<x>-service" directory name -- GCP service account IDs are capped at 30
# characters, and "${environment}-${service_name}-service-account" needs to
# fit under that cap (e.g. "test-storage-service-account" = 29 chars).

variable "project_id" {
  type = string
}

variable "environment" {
  type = string
}

variable "service_name" {
  description = "Short slug for the service, e.g. \"storage\", \"ffmpeg\", \"youtube\", \"telegram\", \"webapp\"."
  type        = string
}

resource "google_service_account" "this" {
  project      = var.project_id
  account_id   = "${var.environment}-${var.service_name}-service-account"
  display_name = "${var.environment} ${var.service_name}-service runtime account"
}

output "email" {
  value = google_service_account.this.email
}

output "member" {
  value = "serviceAccount:${google_service_account.this.email}"
}

output "name" {
  value = google_service_account.this.name
}
