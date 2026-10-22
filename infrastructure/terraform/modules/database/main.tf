resource "aws_dynamodb_table" "users" {
    name = "users-${var.environment}"
    billing_mode = "PAY_PER_REQUEST"
    hash_key = "email"
    
    attribute {
        name = "email"
        type = "S"
    }

    tags = {
        Environment = var.environment
        Project = "expense-tracker"
    }
}

resource "aws_dynamodb_table" "expenses" {
    name = "expenses-${var.environment}"
    billing_mode = "PAY_PER_REQUEST"
    hash_key = "user_id"
    range_key = "sk"

    attribute {
        name = "user_id"
        type = "S"
    }

    attribute {
        name = "sk"
        type = "S"
    }

    tags = {
        Environment = var.environment
        Project = "expense-tracker"
    }
}