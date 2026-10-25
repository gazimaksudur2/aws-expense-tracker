# Staging mirrors dev. Apply only when you intentionally want a second stack
# (extra Cognito MAU / DynamoDB / Lambda usage). Destroy when idle.

locals {
  environment = "staging"
}

module "database" {
  source      = "../../modules/database"
  environment = local.environment
}

module "storage" {
  source      = "../../modules/storage"
  environment = local.environment
}

module "cognito" {
  source      = "../../modules/cognito"
  environment = local.environment
}

module "api" {
  source = "../../modules/api"

  environment          = local.environment
  expenses_table_name  = module.database.expenses_table_name
  expenses_table_arn   = module.database.expenses_table_arn
  receipts_bucket_name = module.storage.bucket_name
  receipts_bucket_arn  = module.storage.bucket_arn
  cognito_issuer       = module.cognito.issuer
  cognito_client_id    = module.cognito.client_id
  api_zip_path         = var.api_zip_path
}

module "monitoring" {
  source = "../../modules/monitoring"

  environment       = local.environment
  api_function_name = module.api.api_function_name
}
