module "enable_apis" {
  source = "../modules/enable-apis"

  project_id = var.project_id
  apis = [
    "storage.googleapis.com",
  ]
}
