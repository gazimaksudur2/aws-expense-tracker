output "role_arn" {
  description = "Set this as the GitHub Actions secret AWS_ROLE_ARN."
  value       = aws_iam_role.github_actions.arn
}

output "oidc_provider_arn" {
  value = aws_iam_openid_connect_provider.github.arn
}
