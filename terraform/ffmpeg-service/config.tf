# Non-secret config.yaml, served from a plain GCS object (not Secret Manager).
resource "google_storage_bucket" "config" {
  project                     = var.project_id
  name                        = "${var.project_id}-${var.service_name}-config"
  location                    = var.region
  uniform_bucket_level_access = true
  force_destroy               = true
}

resource "google_storage_bucket_object" "config" {
  bucket = google_storage_bucket.config.name
  name   = "config.yaml"

  content = templatefile("${path.module}/templates/config.yaml.tftpl", {
    log_level             = var.log_level
    container_port        = var.container_port
    temp_storage_endpoint = var.temp_storage_endpoint
    temp_storage_bucket   = var.temp_storage_bucket
    temp_storage_secure   = var.temp_storage_secure
  })
}

resource "google_storage_bucket_iam_member" "config_viewer" {
  bucket = google_storage_bucket.config.name
  role   = "roles/storage.objectViewer"
  member = "serviceAccount:${local.runtime_service_account}"
}
