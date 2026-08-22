# Grants a role on a bucket, addressed by name (not resource reference) so
# this can be called from a different Terraform state than the one that
# created the bucket (e.g. terraform/global/) with no cross-state coupling.

variable "bucket_name" {
  type = string
}

variable "role" {
  type = string
}

variable "member" {
  description = "IAM member, e.g. \"serviceAccount:...\"."
  type        = string
}

resource "google_storage_bucket_iam_member" "this" {
  bucket = var.bucket_name
  role   = var.role
  member = var.member
}
