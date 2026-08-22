# Shared buckets used by multiple services. Access grants are NOT declared
# here -- each consuming service's own directory grants its own dedicated
# service account access via the bucket-access module, addressed by bucket
# name, so a service's Terraform fully describes its own permissions without
# reading this state.
#
# Pre-existing data buckets, imported into Terraform (not created by it
# originally). Config here mirrors real settings as of import time -- a
# mismatched config will make Terraform try to "fix" real data buckets on
# the next apply, so double check with `terraform plan` after import before
# ever applying.

resource "google_storage_bucket" "temp" {
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

  depends_on = [module.enable_apis]
}

resource "google_storage_bucket" "media" {
  project                     = var.project_id
  name                        = "melo-${var.environment}-media"
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

  depends_on = [module.enable_apis]
}
