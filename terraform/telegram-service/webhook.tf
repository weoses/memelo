# Patches WEBHOOK_EXTERNALURL onto the already-created service, replacing
# the container.tf placeholder with its real (now-known) Cloud Run URL --
# a resource can't reference its own computed attribute within its own
# config (a genuine cycle, not just a staleness problem), so this has to
# happen as a follow-up step, not declaratively. Runs on every apply
# (triggers_replace = [timestamp()]) since a plain `terraform apply` resets
# the env var back to container.tf's placeholder first (Terraform's env
# list is fully authoritative), and this patches it forward again
# immediately after -- end state after any completed apply is always the
# real URL.
resource "terraform_data" "register_webhook_url" {
  triggers_replace = [timestamp()]

  provisioner "local-exec" {
    command = <<-EOT
      set -e
      gcloud run services update ${module.container.name} \
        --project=${var.project_id} --region=${var.region} \
        --update-env-vars=WEBHOOK_EXTERNALURL=${module.container.uri}/webhook \
        --quiet
    EOT
  }
}
