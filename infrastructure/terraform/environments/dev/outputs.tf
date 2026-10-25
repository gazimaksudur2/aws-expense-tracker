output "api_endpoint" {
  description = "Base URL for the HTTP API"
  value       = module.api.api_endpoint
}

output "users_table_name" {
  value = module.database.users_table_name
}

output "expenses_table_name" {
  value = module.database.expenses_table_name
}

output "receipts_bucket_name" {
  value = module.storage.bucket_name
}

output "dashboard_name" {
  value = module.monitoring.dashboard_name
}

output "cognito_user_pool_id" {
  value = module.cognito.user_pool_id
}

output "cognito_client_id" {
  value = module.cognito.client_id
}

output "cognito_issuer" {
  value = module.cognito.issuer
}
