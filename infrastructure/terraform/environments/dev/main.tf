locals {
    environment = "dev"
    jwt_secret_param_name = "/expense-tracker/dev/jwt-secret"
}

module "database" {
    source = "../../modules/database"
    environment = local.environment
}

module "storage" {
    source = "../../modules/storage"
    environment = local.environment
}

module "api" {
    source = "../../modules/api"

    environment = local.environment
    users_table_name = module.database.users_table_name
    users_table_arn = module.database.users_table_arn
    expenses_table_name = module.database.expenses_table_name
    expenses_table_arn = module.database.expenses_table_arn
    receipts_bucket_name = module.storage.bucket_name
    receipts_bucket_arn = module.storage.bucket_arn
    jwt_secret_param_name = local.jwt_secret_param_name
    api_zip_path = var.api_zip_path
    authorizer_zip_path = var.authorizer_zip_path
}

module "monitoring" {
    source = "../../modules/monitoring"

    environment = local.environment
    api_function_name = module.api.api_function_name
    authorizer_function_name = module.api.authorizer_function_name
}