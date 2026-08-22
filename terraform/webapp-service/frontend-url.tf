# See telegram-service/webhook.tf for the full explanation of why this has
# to be an imperative follow-up step rather than a declarative env value.
# Unlike telegram's webhook, this isn't startup-critical -- FRONTEND_BASEURL
# is read per-request from already-served HTML, not at boot -- but it's
# cheap to keep correct the same way.
resource "terraform_data" "patch_frontend_base_url" {
  triggers_replace = [timestamp()]

  provisioner "local-exec" {
    command = <<-EOT
      set -e
      gcloud run services update ${module.container.name} \
        --project=${var.project_id} --region=${var.region} \
        --update-env-vars=FRONTEND_BASEURL=${module.container.uri} \
        --quiet
    EOT
  }
}
