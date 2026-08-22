# ffmpeg-service only touches temp storage (input/output video objects), not
# the media bucket.
module "temp_bucket_access" {
  source = "../modules/bucket-access"

  bucket_name = "melo-${var.environment}-temp"
  role        = "roles/storage.objectAdmin"
  member      = module.service_account.member
}
