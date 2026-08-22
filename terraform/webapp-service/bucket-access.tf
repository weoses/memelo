module "temp_bucket_access" {
  source = "../modules/bucket-access"

  bucket_name = "melo-${var.environment}-temp"
  role        = "roles/storage.objectAdmin"
  member      = module.service_account.member
}
