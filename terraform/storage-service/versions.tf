terraform {
  required_version = ">= 1.5.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
    sops = {
      source  = "carlpett/sops"
      version = "~> 1.0"
    }
    terraform = {
      source = "terraform.io/builtin/terraform"
    }
  }

  # Bucket is intentionally left unset here (partial configuration) since it
  # differs per environment. Supply it at init time, e.g.:
  #   terraform init -backend-config=env/test/backend-test.hcl
  #   terraform init -backend-config=env/prod/backend-prod.hcl
  backend "gcs" {}
}
