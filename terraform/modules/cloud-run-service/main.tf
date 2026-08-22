# A single Cloud Run v2 service with health probes and an optional public
# (allUsers) invoker binding. Grants for specific calling services' service
# accounts are NOT declared here -- each calling service's own directory
# declares the google_cloud_run_v2_service_iam_member it needs on this
# service, addressed by name, keeping "who is allowed to call me" symmetric
# with "who am I allowed to call" and fully described by the caller's own
# state.

variable "project_id" {
  type = string
}

variable "environment" {
  type = string
}

variable "region" {
  type = string
}

variable "service_name" {
  description = "Full Cloud Run service name suffix, e.g. \"ffmpeg-service\" -- the deployed name is environment + \"-\" + service_name."
  type        = string
}

variable "image" {
  type = string
}

variable "container_port" {
  type = number
}

variable "env" {
  description = "Plain (non-secret) container env vars."
  type        = map(string)
  default     = {}
}

variable "secret_env" {
  description = "Container env vars sourced from Secret Manager: name => secret id (as returned by a secret's `id`/`google_secret_manager_secret.*.id`)."
  type        = map(string)
  default     = {}
}

variable "secrets_volume_secret_id" {
  description = "Secret Manager secret id to mount whole as a dotenv file (sourced by the image's entrypoint.sh), or null to skip."
  type        = string
  default     = null
}

variable "secrets_volume_mount_path" {
  description = "Must be at least two path segments below /etc/secrets -- entrypoint.sh sources every file matching /etc/secrets/*/*."
  type        = string
  default     = "/etc/secrets/secrets"
}

variable "cpu" {
  type    = string
  default = "1"
}

variable "memory" {
  type    = string
  default = "512Mi"
}

variable "min_instances" {
  type    = number
  default = 0
}

variable "max_instances" {
  type    = number
  default = 2
}

variable "service_account_email" {
  type = string
}

variable "health_path" {
  type    = string
  default = "/health"
}

variable "ingress" {
  type    = string
  default = "INGRESS_TRAFFIC_ALL"
}

variable "allow_unauthenticated" {
  description = "If true, grants roles/run.invoker to allUsers (public frontend / webhook endpoints). If false (default), only explicit per-caller invoker grants declared elsewhere can invoke this service."
  type        = bool
  default     = false
}

resource "google_cloud_run_v2_service" "this" {
  name                = "${var.environment}-${var.service_name}"
  project             = var.project_id
  location            = var.region
  ingress             = var.ingress
  deletion_protection = false

  template {
    service_account = var.service_account_email

    scaling {
      min_instance_count = var.min_instances
      max_instance_count = var.max_instances
    }

    dynamic "volumes" {
      for_each = var.secrets_volume_secret_id != null ? [var.secrets_volume_secret_id] : []
      content {
        name = "secrets"
        secret {
          secret = volumes.value
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

      startup_probe {
        http_get {
          path = var.health_path
        }
        initial_delay_seconds = 0
        period_seconds        = 5
        timeout_seconds       = 3
        failure_threshold     = 6
      }

      liveness_probe {
        http_get {
          path = var.health_path
        }
        period_seconds    = 30
        timeout_seconds   = 5
        failure_threshold = 3
      }

      dynamic "env" {
        for_each = var.env
        content {
          name  = env.key
          value = env.value
        }
      }

      dynamic "env" {
        for_each = var.secret_env
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

      dynamic "volume_mounts" {
        for_each = var.secrets_volume_secret_id != null ? [1] : []
        content {
          name       = "secrets"
          mount_path = var.secrets_volume_mount_path
        }
      }
    }
  }
}

resource "google_cloud_run_v2_service_iam_member" "public" {
  count = var.allow_unauthenticated ? 1 : 0

  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.this.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

output "uri" {
  value = google_cloud_run_v2_service.this.uri
}

output "name" {
  value = google_cloud_run_v2_service.this.name
}
