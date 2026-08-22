# Informational only. Other modules reference these buckets by the same
# plain "melo-<environment>-temp"/"-media" name strings (deterministic,
# already used throughout the codebase) rather than via
# terraform_remote_state, so this state never needs to be read by another.

output "temp_bucket_name" {
  value = google_storage_bucket.temp.name
}

output "media_bucket_name" {
  value = google_storage_bucket.media.name
}
