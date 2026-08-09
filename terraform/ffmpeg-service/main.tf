resource "google_project_service" "project" {
  project            = var.project_id
  service            = "run.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "secretmanager" {
  project            = var.project_id
  service            = "secretmanager.googleapis.com"
  disable_on_destroy = false
}

data "google_project" "current" {
  project_id = var.project_id
}

resource "terraform_data" "project_initialized" {
  depends_on = [
    google_project_service.project,
    google_project_service.secretmanager,
    data.google_project.current
  ]
}
