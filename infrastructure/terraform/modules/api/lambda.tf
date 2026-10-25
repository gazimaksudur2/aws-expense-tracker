resource "aws_lambda_function" "api" {
  function_name = "expense-tracker-api-${var.environment}"
  role          = aws_iam_role.api.arn
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  architectures = ["x86_64"]
  filename      = var.api_zip_path
  source_code_hash = filebase64sha256(var.api_zip_path)
  timeout       = 10
  memory_size   = 128

  environment {
    variables = {
      APP_MODE        = "lambda"
      USERS_TABLE     = var.users_table_name
      EXPENSES_TABLE  = var.expenses_table_name
      RECEIPTS_BUCKET = var.receipts_bucket_name
      # Phase 1 login still signs JWTs in the business function.
      # Injected at deploy time; the role policy has no ssm:GetParameter.
      JWT_SECRET = data.aws_ssm_parameter.jwt_secret.value
    }
  }
}

resource "aws_lambda_function" "authorizer" {
  function_name = "expense-tracker-authorizer-${var.environment}"
  role          = aws_iam_role.authorizer.arn
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  architectures = ["x86_64"]
  filename      = var.authorizer_zip_path
  source_code_hash = filebase64sha256(var.authorizer_zip_path)
  timeout       = 5
  memory_size   = 128

  environment {
    variables = {
      JWT_SECRET = data.aws_ssm_parameter.jwt_secret.value
    }
  }
}

resource "aws_lambda_permission" "api_invoke" {
  statement_id  = "AllowAPIGatewayInvokeAPI"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.api.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.http.execution_arn}/*/*"
}

resource "aws_lambda_permission" "authorizer_invoke" {
  statement_id  = "AllowAPIGatewayInvokeAuthorizer"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.authorizer.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.http.execution_arn}/authorizers/${aws_apigatewayv2_authorizer.jwt.id}"
}