# Decrypted locally by the sops provider at plan/apply time (using whatever
# key sops is configured with, e.g. GCP KMS/age) -- only the ciphertext in
# env/<environment>/secrets.enc.env ever touches disk or git.
data "sops_file" "secrets" {
  source_file = "env/${var.environment}/secrets.enc.env"
  input_type  = "dotenv"
}

# Secret Manager secret backing "${environment}-temp-storage-secrets" in var.secret_names.
# Unlike other entries in secret_names (still expected to be created out of
# band), Terraform owns this one end to end: it creates the secret and writes
# its content from the decrypted sops file below. main.tf's files_accessor /
# volume wiring picks it up the same way it would an externally-created one.
resource "google_secret_manager_secret" "secrets" {
  project   = var.project_id
  secret_id = "${var.environment}-secrets"

  replication {
    auto {}
  }

  depends_on = [google_project_service.secretmanager]
}

resource "google_secret_manager_secret_version" "secrets" {
  secret      = google_secret_manager_secret.secrets.id
  secret_data = data.sops_file.secrets.raw
}

resource "terraform_data" "secrets_initialized" {
  depends_on = [
    data.sops_file.secrets,
    google_secret_manager_secret.secrets,
    google_secret_manager_secret_version.secrets,
  ]
}
