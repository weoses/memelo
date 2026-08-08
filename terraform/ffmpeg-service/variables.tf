variable "project_id" {
  description = "GCP project ID to deploy into"
  type        = string
}

variable "region" {
  description = "GCP region for the Cloud Run service"
  type        = string
  default     = "us-central1"
}

variable "service_name" {
  description = "Cloud Run service name"
  type        = string
  default     = "ffmpeg-service"
}

variable "image" {
  description = "Container image to deploy, e.g. ghcr.io/weoses/ffmpeg-service:latest"
  type        = string
}

variable "container_port" {
  description = "Port the container listens on (also used to set SERVER_LISTENADDRESS)"
  type        = number
  default     = 8080
}

variable "cpu" {
  description = "CPU limit for the container"
  type        = string
  default     = "1"
}

variable "memory" {
  description = "Memory limit for the container"
  type        = string
  default     = "512Mi"
}

variable "min_instances" {
  description = "Minimum number of Cloud Run instances"
  type        = number
  default     = 0
}

variable "max_instances" {
  description = "Maximum number of Cloud Run instances"
  type        = number
  default     = 2
}

variable "temp_storage_endpoint" {
  description = "S3/MinIO-compatible endpoint used for temp media storage"
  type        = string
}

variable "temp_storage_bucket" {
  description = "Bucket name used for temp media storage"
  type        = string
  default     = "melo-temp"
}

variable "temp_storage_secure" {
  description = "Whether the temp storage endpoint uses TLS"
  type        = bool
  default     = false
}

variable "secret_names" {
  description = <<-EOT
    Names (Secret Manager secret IDs) of secrets to mount into the container,
    e.g. ["temp-storage-secrets"]. These secrets must already exist in Secret
    Manager, created and populated outside Terraform (e.g. `gcloud secrets
    create <name> --data-file=...`), typically as KEY=VALUE lines -- Terraform
    never creates, writes, or reads their content, only grants access and
    mounts them. If a name here doesn't already exist as a secret, apply fails.
  EOT
  type        = list(string)
  default     = []
}

variable "invoker_members" {
  description = "IAM members granted roles/run.invoker on the service (e.g. [\"serviceAccount:storage-service@<project>.iam.gserviceaccount.com\"]). Empty by default; the service is not publicly invokable."
  type        = list(string)
  default     = []
}

variable "log_level" {
  description = "Log level written into the mounted config.yaml"
  type        = string
  default     = "info"
}

variable "service_account_email" {
  description = "Runtime service account email for the Cloud Run service. If null, Cloud Run uses the project's default compute service account, which is also what gets granted access to the mounted config secret."
  type        = string
  default     = null
}
