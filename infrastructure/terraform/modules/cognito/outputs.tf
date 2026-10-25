output "user_pool_id" {
  value = aws_cognito_user_pool.main.id
}

output "user_pool_arn" {
  value = aws_cognito_user_pool.main.arn
}

output "user_pool_endpoint" {
  description = "Issuer host without https:// (cognito-idp.REGION.amazonaws.com/POOL_ID)."
  value       = aws_cognito_user_pool.main.endpoint
}

output "client_id" {
  value = aws_cognito_user_pool_client.app.id
}

output "issuer" {
  description = "Full issuer URL for the API Gateway JWT authorizer."
  value       = "https://${aws_cognito_user_pool.main.endpoint}"
}
