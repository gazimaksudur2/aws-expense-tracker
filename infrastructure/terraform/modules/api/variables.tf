variable "environment" {
  description = "Environment name used in resource names (dev, staging, prod)"
  type        = string
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

variable "cognito_issuer" {
  description = "Cognito user pool issuer URL (https://cognito-idp.REGION.amazonaws.com/POOL_ID)."
  type        = string
}

variable "cognito_client_id" {
  description = "Cognito app client ID (JWT audience)."
  type        = string
}

variable "api_zip_path" {
  description = "Path to the business Lambda zip (bootstrap binary inside)."
  type        = string
}
