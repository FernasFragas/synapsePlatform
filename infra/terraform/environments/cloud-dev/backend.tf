# Keep local state by default for this skeleton.
# Configure a real remote backend in your deployment workflow, not with
# account-specific values committed to the repository.
#
# Example: S3 backend with DynamoDB locking
#
# terraform {
#   backend "s3" {
#     bucket         = "replace-with-terraform-state-bucket"
#     key            = "synapse-platform/cloud-dev/terraform.tfstate"
#     region         = "replace-with-region"
#     dynamodb_table = "replace-with-lock-table"
#     encrypt        = true
#   }
# }
#
# Example: GCS backend
#
# terraform {
#   backend "gcs" {
#     bucket = "replace-with-terraform-state-bucket"
#     prefix = "synapse-platform/cloud-dev"
#   }
# }
