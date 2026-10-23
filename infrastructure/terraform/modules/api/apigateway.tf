resource "aws_apigatewayv2_api" "http" {
  name          = "expense-tracker-${var.environment}"
  protocol_type = "HTTP"

  tags = {
    Environment = var.environment
    Project     = "expense-tracker"
  }
}

resource "aws_apigatewayv2_authorizer" "jwt" {
  api_id                            = aws_apigatewayv2_api.http.id
  authorizer_type                   = "REQUEST"
  authorizer_uri                    = aws_lambda_function.authorizer.invoke_arn
  identity_sources                  = ["$request.header.Authorization"]
  name                              = "jwt"
  authorizer_payload_format_version = "2.0"
  enable_simple_responses           = true
  authorizer_result_ttl_in_seconds  = 0
}

resource "aws_apigatewayv2_integration" "api" {
  api_id                 = aws_apigatewayv2_api.http.id
  integration_type       = "AWS_PROXY"
  integration_uri        = aws_lambda_function.api.invoke_arn
  payload_format_version = "2.0"
}

# Public routes
resource "aws_apigatewayv2_route" "register" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "POST /users/register"
  target    = "integrations/${aws_apigatewayv2_integration.api.id}"
}

resource "aws_apigatewayv2_route" "login" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "POST /users/login"
  target    = "integrations/${aws_apigatewayv2_integration.api.id}"
}

# Protected routes
resource "aws_apigatewayv2_route" "create_expense" {
  api_id             = aws_apigatewayv2_api.http.id
  route_key          = "POST /expenses"
  target             = "integrations/${aws_apigatewayv2_integration.api.id}"
  authorization_type = "CUSTOM"
  authorizer_id      = aws_apigatewayv2_authorizer.jwt.id
}

resource "aws_apigatewayv2_route" "monthly_expenses" {
  api_id             = aws_apigatewayv2_api.http.id
  route_key          = "GET /expenses/monthly"
  target             = "integrations/${aws_apigatewayv2_integration.api.id}"
  authorization_type = "CUSTOM"
  authorizer_id      = aws_apigatewayv2_authorizer.jwt.id
}

resource "aws_apigatewayv2_route" "analytics" {
  api_id             = aws_apigatewayv2_api.http.id
  route_key          = "GET /analytics"
  target             = "integrations/${aws_apigatewayv2_integration.api.id}"
  authorization_type = "CUSTOM"
  authorizer_id      = aws_apigatewayv2_authorizer.jwt.id
}

resource "aws_apigatewayv2_route" "receipts_upload" {
  api_id             = aws_apigatewayv2_api.http.id
  route_key          = "POST /receipts/upload"
  target             = "integrations/${aws_apigatewayv2_integration.api.id}"
  authorization_type = "CUSTOM"
  authorizer_id      = aws_apigatewayv2_authorizer.jwt.id
}

resource "aws_apigatewayv2_stage" "default" {
  api_id      = aws_apigatewayv2_api.http.id
  name        = "$default"
  auto_deploy = true
}