variable "project_id" {
  description = "GCP project ID to deploy into"
  type        = string
}

variable "environment" {
  description = "environment (test/prod/etc)"
  type        = string
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
  default     = "2"
}

variable "memory" {
  description = "Memory limit for the container"
  type        = string
  default     = "1Gi"
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

variable "embedding_project_name" {
  description = "value for media-embedding.ProjectName / audio-stt.ProjectName"
  type        = string
  default     = ""
}

variable "extractor_provider" {
  description = "gemini | openrouter"
  type        = string
  default     = "gemini"
}

variable "embedder_provider" {
  description = "gemini | openrouter"
  type        = string
  default     = "gemini"
}
