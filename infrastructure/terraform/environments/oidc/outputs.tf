output "aws_role_arn" {
  description = "Add this value as a GitHub Actions repository secret named AWS_ROLE_ARN."
  value       = module.github_oidc.role_arn
}
