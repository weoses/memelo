# Overrides the values baked into the image's default config.yaml
# (ffmpeg-service/config.yaml) via Viper's AutomaticEnv, same mechanism
# used for the mounted secret files below.
locals {
  container_env = {
    SERVER_LISTENADDRESS  = ":7003"
    TEMP_STORAGE_ENDPOINT = var.google_storage_host
    TEMP_STORAGE_BUCKET   = google_storage_bucket.temp_bucket.name
    TEMP_STORAGE_SECURE   = "true"
  }

  container_secret_env = {
    TEMP_STORAGE_ACCESSKEY = google_secret_manager_secret.service_account_hmac_akey.id
    TEMP_STORAGE_SECRETKEY = google_secret_manager_secret.service_account_hmac_skey.id
  }
}

resource "google_cloud_run_v2_service" "ffmpeg_service" {
  name                = "${var.environment}-ffmpeg-service"
  project             = var.project_id
  location            = var.region
  ingress             = "INGRESS_TRAFFIC_ALL"
  deletion_protection = false

  depends_on = [
    terraform_data.project_initialized,
    terraform_data.buckets_initialized,
    terraform_data.secrets_initialized,
    terraform_data.service_account_initialized
  ]

  template {
    service_account = google_service_account.service_account.email

    scaling {
      min_instance_count = var.min_instances
      max_instance_count = var.max_instances
    }

    volumes {
      name = "secrets"
      secret {
        secret = google_secret_manager_secret.secrets.secret_id
      }
    }

    containers {
      image = "ghcr.io/weoses/ffmpeg-service:${var.image_tag}"

      ports {
        container_port = 7003
      }

      resources {
        limits = {
          cpu    = var.cpu
          memory = var.memory
        }
      }

      startup_probe {
        http_get {
          path = "/health"
        }
        initial_delay_seconds = 0
        period_seconds        = 5
        timeout_seconds       = 3
        failure_threshold     = 6
      }

      liveness_probe {
        http_get {
          path = "/health"
        }
        period_seconds    = 30
        timeout_seconds   = 5
        failure_threshold = 3
      }

      dynamic "env" {
        for_each = local.container_env
        content {
          name  = env.key
          value = env.value
        }
      }

      dynamic "env" {
        for_each = local.container_secret_env
        content {
          name = env.key
          value_source {
            secret_key_ref {
              secret  = env.value
              version = "latest"
            }
          }
        }
      }
      # entrypoint.sh sources every file matching /etc/secrets/*/*, so this
      # must land the secret at least two path segments below /etc/secrets.
      volume_mounts {
        name       = "secrets"
        mount_path = "/etc/secrets/secrets"
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
