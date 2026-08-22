# A Secret Manager secret whose content is Terraform-owned end to end,
# sourced from a sops+PGP-encrypted dotenv file. Only ciphertext ever touches
# disk or git -- plaintext exists only in Terraform's in-memory plan and in
# the resulting Secret Manager version (and, as with any secret value
# Terraform manages, in the remote state file).
#
# source_file is resolved relative to the current working directory of the
# `terraform` process (not this module's directory) -- always run terraform
# from inside the calling service's own directory, e.g.
# `cd terraform/storage-service && terraform apply`, matching the existing
# convention.

variable "project_id" {
  type = string
}

variable "secret_id" {
  type = string
}

variable "source_file" {
  description = "Path to the sops-encrypted dotenv file, e.g. \"env/test/secrets.enc.env\"."
  type        = string
}

variable "accessor_member" {
  description = "IAM member (e.g. \"serviceAccount:...\") granted roles/secretmanager.secretAccessor on this secret."
  type        = string
}

data "sops_file" "this" {
  source_file = var.source_file
  input_type  = "dotenv"
}

resource "google_secret_manager_secret" "this" {
  project   = var.project_id
  secret_id = var.secret_id

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "this" {
  secret      = google_secret_manager_secret.this.id
  secret_data = data.sops_file.this.raw
}

resource "google_secret_manager_secret_iam_member" "this" {
  member    = var.accessor_member
  role      = "roles/secretmanager.secretAccessor"
  secret_id = google_secret_manager_secret.this.id
}

output "secret_id" {
  value = google_secret_manager_secret.this.secret_id
}

output "id" {
  value = google_secret_manager_secret.this.id
}
