variable "project_id" {
  description = "GCP project ID to deploy into"
  type        = string
}

variable "environment" {
  description = "environment (test/prod/etc)"
  type        = string
}

variable "region" {
  description = "GCP region for the buckets"
  type        = string
  default     = "us-central1"
}
