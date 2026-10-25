resource "aws_lambda_function" "api" {
  function_name    = "expense-tracker-api-${var.environment}"
  role             = aws_iam_role.api.arn
  handler          = "bootstrap"
  runtime          = "provided.al2023"
  architectures    = ["x86_64"]
  filename         = var.api_zip_path
  source_code_hash = filebase64sha256(var.api_zip_path)
  timeout          = 10
  memory_size      = 128

  environment {
    variables = {
      APP_MODE        = "lambda"
      EXPENSES_TABLE  = var.expenses_table_name
      RECEIPTS_BUCKET = var.receipts_bucket_name
      # No JWT_SECRET: API Gateway JWT authorizer validates Cognito; Lambda reads `sub` from context.
      # USERS_TABLE omitted: Cognito owns identity (local Docker still uses register/login + users table).
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
