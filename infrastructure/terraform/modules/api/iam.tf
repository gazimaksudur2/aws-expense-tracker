data "aws_ssm_parameter" "jwt_secret" {
    name = var.jwt_secret_param_name
    with_decryption = true
}

data "aws_caller_identity" "current" {}
data "aws_region" "current" {}

resource "aws_iam_role" "api" {
    name = "expense-tracker-api-${var.environment}"

    assume_role_policy = jsonencode({
        Version = "2012-10-17"
        Statement = [{
            Effect = "Allow"
            Principal = { Service = "lambda.amazonaws.com" }
            Action = "sts:AssumeRole"
        }]
    })
}

resource "aws_iam_role_policy" "api" {
    name = "expense-tracker-api-${var.environment}"
    role = aws_iam_role.api.id
    
    policy = jsonencode({
        Version = "2012-10-17"
        Statement = [
            {
                Sid = "DynamoDB"
                Effect = "Allow"
                Action = [
                    "dynamodb:GetItem",
                    "dynamodb:PutItem",
                    "dynamodb:Query",
                ]
                Resource = [
                    var.users_table_arn,
                    var.expenses_table_arn,
                ]
            },
            {
                Sid = "S3Receipts"
                Effect = "Allow"
                Action = [
                    "s3:PutObject",
                    "s3:GetObject",
                ]
                resource = "${var.receipts_bucket_arn}/receipts/*"
            },
            {
                Sid = "Logs"
                Effect = "Allow"
                Action = [
                    "logs:CreateLogStream",
                    "logs:PutLogEvents"
                ]
                Resource = "arn:aws:logs:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:log-group:/aws/lambda/expense-tracker-api-${var.environment}:*"
            },
        ]
    })
}

resource "aws_iam_role" "authorizer" {
  name = "expense-tracker-authorizer-${var.environment}"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "lambda.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}
resource "aws_iam_role_policy" "authorizer" {
  name = "expense-tracker-authorizer-${var.environment}"
  role = aws_iam_role.authorizer.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "ReadJWTSecret"
        Effect   = "Allow"
        Action   = ["ssm:GetParameter"]
        Resource = data.aws_ssm_parameter.jwt_secret.arn
      },
      {
        Sid      = "Logs"
        Effect   = "Allow"
        Action   = ["logs:CreateLogStream", "logs:PutLogEvents"]
        Resource = "arn:aws:logs:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:log-group:/aws/lambda/expense-tracker-authorizer-${var.environment}:*"
      },
    ]
  })
}
