resource "google_service_account" "service_account" {
  account_id   = "${var.environment}-runtime-service-account"
  display_name = "[${var.environment}] Runtime service account"
}


resource "google_storage_hmac_key" "gcs_key" {
  service_account_email = google_service_account.service_account.email
}

resource "terraform_data" "service_account_initialized" {
  depends_on = [
    google_service_account.service_account,
    google_storage_hmac_key.gcs_key
  ]
}
