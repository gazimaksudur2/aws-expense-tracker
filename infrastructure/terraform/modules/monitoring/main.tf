resource "aws_cloudwatch_log_group" "api" {
  name              = "/aws/lambda/${var.api_function_name}"
  retention_in_days = 14

  tags = {
    Environment = var.environment
    Project     = "expense-tracker"
  }
}

resource "aws_cloudwatch_log_group" "authorizer" {
  name              = "/aws/lambda/${var.authorizer_function_name}"
  retention_in_days = 14

  tags = {
    Environment = var.environment
    Project     = "expense-tracker"
  }
}

resource "aws_cloudwatch_metric_alarm" "api_errors" {
  alarm_name          = "expense-tracker-${var.environment}-api-errors"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  metric_name         = "Errors"
  namespace           = "AWS/Lambda"
  period              = 60
  statistic           = "Sum"
  threshold           = 0
  alarm_description   = "Business Lambda reported one or more errors in the last minute."
  treat_missing_data  = "notBreaching"

  dimensions = {
    FunctionName = var.api_function_name
  }

  tags = {
    Environment = var.environment
    Project     = "expense-tracker"
  }
}

resource "aws_cloudwatch_dashboard" "main" {
  dashboard_name = "expense-tracker-${var.environment}"

  dashboard_body = jsonencode({
    widgets = [
      {
        type   = "metric"
        x      = 0
        y      = 0
        width  = 12
        height = 6
        properties = {
          title  = "Lambda invocations"
          region = data.aws_region.current.name
          metrics = [
            ["AWS/Lambda", "Invocations", "FunctionName", var.api_function_name, { label = "api" }],
            [".", "Invocations", ".", var.authorizer_function_name, { label = "authorizer" }],
          ]
          stat   = "Sum"
          period = 300
        }
      },
      {
        type   = "metric"
        x      = 12
        y      = 0
        width  = 12
        height = 6
        properties = {
          title  = "Lambda errors"
          region = data.aws_region.current.name
          metrics = [
            ["AWS/Lambda", "Errors", "FunctionName", var.api_function_name, { label = "api" }],
            [".", "Errors", ".", var.authorizer_function_name, { label = "authorizer" }],
          ]
          stat   = "Sum"
          period = 300
        }
      },
      {
        type   = "metric"
        x      = 0
        y      = 6
        width  = 24
        height = 6
        properties = {
          title  = "API latency (p50 / p99)"
          region = data.aws_region.current.name
          metrics = [
            ["AWS/Lambda", "Duration", "FunctionName", var.api_function_name, { stat = "p50", label = "p50" }],
            ["...", { stat = "p99", label = "p99" }],
          ]
          period = 300
        }
      }
    ]
  })
}

data "aws_region" "current" {}