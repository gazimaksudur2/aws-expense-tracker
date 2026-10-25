resource "aws_apigatewayv2_api" "http" {
  name          = "expense-tracker-${var.environment}"
  protocol_type = "HTTP"

  tags = {
    Environment = var.environment
    Project     = "expense-tracker"
  }
}

# Built-in JWT authorizer (Cognito ID tokens). No Lambda authorizer.
resource "aws_apigatewayv2_authorizer" "cognito" {
  api_id           = aws_apigatewayv2_api.http.id
  authorizer_type  = "JWT"
  identity_sources = ["$request.header.Authorization"]
  name             = "cognito"

  jwt_configuration {
    audience = [var.cognito_client_id]
    issuer   = var.cognito_issuer
  }
}

resource "aws_apigatewayv2_integration" "api" {
  api_id                 = aws_apigatewayv2_api.http.id
  integration_type       = "AWS_PROXY"
  integration_uri        = aws_lambda_function.api.invoke_arn
  payload_format_version = "2.0"
}

# Protected routes — Cognito owns register/login; those routes are not on the API.
resource "aws_apigatewayv2_route" "create_expense" {
  api_id             = aws_apigatewayv2_api.http.id
  route_key          = "POST /expenses"
  target             = "integrations/${aws_apigatewayv2_integration.api.id}"
  authorization_type = "JWT"
  authorizer_id      = aws_apigatewayv2_authorizer.cognito.id
}

resource "aws_apigatewayv2_route" "monthly_expenses" {
  api_id             = aws_apigatewayv2_api.http.id
  route_key          = "GET /expenses/monthly"
  target             = "integrations/${aws_apigatewayv2_integration.api.id}"
  authorization_type = "JWT"
  authorizer_id      = aws_apigatewayv2_authorizer.cognito.id
}

resource "aws_apigatewayv2_route" "analytics" {
  api_id             = aws_apigatewayv2_api.http.id
  route_key          = "GET /analytics"
  target             = "integrations/${aws_apigatewayv2_integration.api.id}"
  authorization_type = "JWT"
  authorizer_id      = aws_apigatewayv2_authorizer.cognito.id
}

resource "aws_apigatewayv2_route" "receipts_upload" {
  api_id             = aws_apigatewayv2_api.http.id
  route_key          = "POST /receipts/upload"
  target             = "integrations/${aws_apigatewayv2_integration.api.id}"
  authorization_type = "JWT"
  authorizer_id      = aws_apigatewayv2_authorizer.cognito.id
}

resource "aws_apigatewayv2_stage" "default" {
  api_id      = aws_apigatewayv2_api.http.id
  name        = "$default"
  auto_deploy = true
}
