resource "google_project_service" "run" {
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

# Runtime identity Cloud Run will actually use, for granting secret/storage access.
locals {
  runtime_service_account = coalesce(
    var.service_account_email,
    "${data.google_project.current.number}-compute@developer.gserviceaccount.com",
  )

  secret_names = toset(var.secret_names)
  # Cloud Run volume names must be lowercase RFC1123 labels; secret IDs may
  # contain underscores, so sanitize for the volume/mount name while keeping
  # the secret_id reference itself untouched.
  secret_volume_name = { for name in local.secret_names : name => lower(replace(name, "_", "-")) }
}

# Secret files: opaque named blobs (KEY=VALUE lines) mounted into the container
# and sourced by the entrypoint script before the app starts. These secrets are
# created and populated OUTSIDE Terraform (e.g. `gcloud secrets create` /
# console) -- Terraform only references them by name to grant access and mount
# them. It never creates a secret or writes a version. If a name in
# var.secret_names doesn't already exist in Secret Manager, apply fails.
resource "google_secret_manager_secret_iam_member" "files_accessor" {
  for_each = local.secret_names

  project    = var.project_id
  secret_id  = each.value
  role       = "roles/secretmanager.secretAccessor"
  member     = "serviceAccount:${local.runtime_service_account}"
  depends_on = [google_project_service.secretmanager]
}

resource "google_cloud_run_v2_service" "ffmpeg_service" {
  name                = var.service_name
  project             = var.project_id
  location            = var.region
  ingress             = "INGRESS_TRAFFIC_ALL"
  deletion_protection = false

  depends_on = [
    google_project_service.run,
    google_storage_bucket_iam_member.config_viewer,
    google_secret_manager_secret_iam_member.files_accessor,
  ]

  template {
    service_account = var.service_account_email

    scaling {
      min_instance_count = var.min_instances
      max_instance_count = var.max_instances
    }

    volumes {
      name = "config"
      gcs {
        bucket    = google_storage_bucket.config.name
        read_only = true
      }
    }

    dynamic "volumes" {
      for_each = local.secret_names
      content {
        name = local.secret_volume_name[volumes.value]
        secret {
          secret = volumes.value
          items {
            version = "latest"
            path    = volumes.value
          }
        }
      }
    }

    containers {
      image = var.image

      ports {
        container_port = var.container_port
      }

      resources {
        limits = {
          cpu    = var.cpu
          memory = var.memory
        }
      }

      volume_mounts {
        name       = "config"
        mount_path = "/app/config"
      }

      dynamic "volume_mounts" {
        for_each = local.secret_names
        content {
          name       = local.secret_volume_name[volume_mounts.value]
          mount_path = "/etc/secrets/${local.secret_volume_name[volume_mounts.value]}"
        }
      }
    }
  }
}

resource "google_cloud_run_v2_service_iam_member" "invoker" {
  for_each = toset(var.invoker_members)

  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.ffmpeg_service.name
  role     = "roles/run.invoker"
  member   = each.value
}
