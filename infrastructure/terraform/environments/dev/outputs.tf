output "api_endpoint" {
    description = "Base URL for the HTTP API"
    value = module.api.api_endpoint
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