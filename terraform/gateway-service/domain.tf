# Pre-existing resource, created outside Terraform -- must be imported
# before the first apply against this file:
#   terraform import google_cloud_run_domain_mapping.this <import-id>
# (confirm the exact import ID format against the google provider docs at
# apply time). It was originally routed at test-webapp-service directly;
# this apply re-points it at the gateway, the new single public entry point.
resource "google_cloud_run_domain_mapping" "this" {
  location = var.region
  name     = var.domain_name

  metadata {
    namespace = var.project_id
  }

  spec {
    route_name = module.container.name
  }
}
