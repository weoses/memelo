# Enables a set of GCP APIs. Safe to call from more than one service's state
# for the same project -- enabling an already-enabled API is a no-op.

variable "project_id" {
  type = string
}

variable "apis" {
  description = "API service names to enable, e.g. [\"run.googleapis.com\"]"
  type        = list(string)
}

resource "google_project_service" "this" {
  for_each = toset(var.apis)

  project            = var.project_id
  service            = each.value
  disable_on_destroy = false
}

resource "terraform_data" "initialized" {
  depends_on = [google_project_service.this]
}

output "initialized" {
  value = terraform_data.initialized.id
}
