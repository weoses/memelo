# storage-service owns/reads both buckets: temp (pipeline intermediates) and
# media (final stored memes).
module "temp_bucket_access" {
  source = "../modules/bucket-access"

  bucket_name = "melo-${var.environment}-temp"
  role        = "roles/storage.objectAdmin"
  member      = module.service_account.member
}

module "media_bucket_access" {
  source = "../modules/bucket-access"

  bucket_name = "melo-${var.environment}-media"
  role        = "roles/storage.objectAdmin"
  member      = module.service_account.member
}
