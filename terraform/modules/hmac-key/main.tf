# A GCS HMAC key owned by a given service account, stored as two Secret
# Manager secrets (access key / secret key) for use as TEMP_STORAGE_ACCESSKEY
# / TEMP_STORAGE_SECRETKEY. Each service that talks to GCS via the S3-HMAC
# interface gets its own HMAC key tied to its own dedicated service account
# -- GCS HMAC signature validation is scoped to the owning SA's own IAM
# grants, not just possession of the secret.

variable "project_id" {
  type = string
}

variable "environment" {
  type = string
}

variable "service_name" {
  description = "Short slug for the service, e.g. \"storage\", \"ffmpeg\"."
  type        = string
}

variable "service_account_email" {
  type = string
}

variable "accessor_member" {
  description = "IAM member (e.g. \"serviceAccount:...\") granted roles/secretmanager.secretAccessor on both secrets -- normally the same SA that owns the HMAC key."
  type        = string
}

resource "google_storage_hmac_key" "this" {
  service_account_email = var.service_account_email
}

resource "google_secret_manager_secret" "access_key" {
  project   = var.project_id
  secret_id = "${var.environment}-${var.service_name}-hmac-accesskey"

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret" "secret_key" {
  project   = var.project_id
  secret_id = "${var.environment}-${var.service_name}-hmac-secretkey"

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "access_key" {
  secret      = google_secret_manager_secret.access_key.id
  secret_data = google_storage_hmac_key.this.access_id
}

resource "google_secret_manager_secret_version" "secret_key" {
  secret      = google_secret_manager_secret.secret_key.id
  secret_data = google_storage_hmac_key.this.secret
}

resource "google_secret_manager_secret_iam_member" "access_key" {
  member    = var.accessor_member
  role      = "roles/secretmanager.secretAccessor"
  secret_id = google_secret_manager_secret.access_key.id
}

resource "google_secret_manager_secret_iam_member" "secret_key" {
  member    = var.accessor_member
  role      = "roles/secretmanager.secretAccessor"
  secret_id = google_secret_manager_secret.secret_key.id
}

output "access_key_secret_id" {
  value = google_secret_manager_secret.access_key.id
}

output "secret_key_secret_id" {
  value = google_secret_manager_secret.secret_key.id
}
