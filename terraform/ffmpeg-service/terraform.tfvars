# Copy to terraform.tfvars and fill in real values.
# terraform.tfvars is gitignored — never commit real secrets.

project_id  = "positive-shell-448806-f7"
region      = "us-central1"
image_tag   = "v00.000.00-35-dev"
environment = "test"
cpu         = "4"
memory      = "4Gi"