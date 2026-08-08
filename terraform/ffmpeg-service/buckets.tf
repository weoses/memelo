# Pre-existing data buckets, imported into Terraform (not created by it
# originally). Config here mirrors real settings as of import time -- see
# README's "Importing pre-existing buckets" section before changing anything,
# since a mismatched config will make Terraform try to "fix" real data buckets
# on the next apply. No force_destroy: these hold real data, unlike the
# ffmpeg-service config bucket in config.tf.

resource "google_storage_bucket" "temp" {
  project                     = var.project_id
  name                        = "melo-test-temp"
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

resource "google_storage_bucket" "media" {
  project                     = var.project_id
  name                        = "melo-test-media"
  location                    = var.region
  storage_class               = "STANDARD"
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"
}
