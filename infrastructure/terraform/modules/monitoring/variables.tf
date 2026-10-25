variable "environment" {
  description = "Environment name used in resource names (dev, staging, prod)."
  type        = string
}

variable "api_function_name" {
  description = "Business Lambda function name."
  type        = string
}
