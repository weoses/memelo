resource "google_service_account" "service_account" {
  account_id   = "${var.environment}-runtime-service-account"
  display_name = "[${var.environment}] Runtime service account"
}




resource "terraform_data" "service_account_initialized" {
  depends_on = [
    google_service_account.service_account,
  ]
}
