output "api_log_group_name" {
  value = aws_cloudwatch_log_group.api.name
}

output "api_log_group_arn" {
  value = aws_cloudwatch_log_group.api.arn
}

output "authorizer_log_group_name" {
  value = aws_cloudwatch_log_group.authorizer.name
}

output "authorizer_log_group_arn" {
  value = aws_cloudwatch_log_group.authorizer.arn
}

output "dashboard_name" {
  value = aws_cloudwatch_dashboard.main.dashboard_name
}