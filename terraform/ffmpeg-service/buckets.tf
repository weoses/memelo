# Pre-existing data buckets, imported into Terraform (not created by it
# originally). Config here mirrors real settings as of import time -- see
# README's "Importing pre-existing buckets" section before changing anything,
# since a mismatched config will make Terraform try to "fix" real data buckets
# on the next apply.

resource "google_storage_bucket" "temp_bucket" {
  project                     = var.project_id
  name                        = "melo-${var.environment}-temp"
  location                    = var.region
  storage_class               = "STANDARD"
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"

  cors {
    origin          = ["*"]
    method          = ["OPTIONS", "GET", "POST", "PUT"]
    response_header = ["Content-Type", "Content-Length"]
    max_age_seconds = 3600
  }

  lifecycle_rule {
    condition {
      age = 10
    }
    action {
      type = "Delete"
    }
  }
}

resource "google_storage_bucket" "media_bucket" {
  project                     = var.project_id
  name                        = "melo-${var.environment}-media"
  location                    = var.region
  storage_class               = "STANDARD"
  uniform_bucket_level_access = false
  public_access_prevention    = "enforced"

  cors {
    origin          = ["*"]
    method          = ["OPTIONS", "GET", "POST", "PUT"]
    response_header = ["Content-Type", "Content-Length"]
    max_age_seconds = 3600
  }
}

resource "google_storage_bucket_iam_member" "media_bucket" {
  bucket = google_storage_bucket.media_bucket.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.service_account.email}"
}

resource "google_storage_bucket_iam_member" "temp_bucket" {
  bucket = google_storage_bucket.temp_bucket.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.service_account.email}"
}

resource "terraform_data" "buckets_initialized" {
  depends_on = [
    google_storage_bucket.temp_bucket,
    google_storage_bucket.media_bucket,
    google_storage_bucket_iam_member.temp_bucket,
    google_storage_bucket_iam_member.media_bucket,
  ]
}