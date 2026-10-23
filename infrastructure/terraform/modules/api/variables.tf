variable "environment" {
    description = "Environment name used in resource names (dev, staging, prod)"
    type = string
}

variable "users_table_name" {
    type = string
}

variable "users_table_arn" {
    type = string
}

variable "expenses_table_name" {
    type = string
}

variable "expenses_table_arn" {
    type = string
}

variable "receipts_bucket_name" {
    type = string
}

variable "receipts_bucket_arn" {
    type = string
}

variable "jwt_secret_param_name" {
    description = "SSM parameter name for the Jwt signing secret."
    type = string
}

variable "api_zip_path" {
    description = "Path to the business Lambda zip (bootstrap binary inside)."
    type = string
}

variable "authorizer_zip_path" {
    description = "Path to the authorizer Lambda zip (bootstrap binary inside)."
    type = string
}