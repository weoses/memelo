variable "project_id" {
  description = "GCP project ID to deploy into"
  type        = string
}

variable "environment" {
  description = "environment (test/prod/etc)"
  type        = string
}

variable "google_storage_host" {
  description = "host of google s3 storage"
  type        = string
  default     = "storage.googleapis.com"
}

variable "region" {
  description = "GCP region for the Cloud Run service"
  type        = string
  default     = "us-central1"
}

variable "image_tag" {
  description = "image tag to deploy"
  type        = string
  default     = "latest"
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
