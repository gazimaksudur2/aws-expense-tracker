output "users_table_name" {
    value = aws_dynamodb_table.users.name
}

output "users_table_arn" {
    value = aws_dynamodb_table.users.arn
}

output "expenses_table_name" {
    value = aws_dynamodb_table.expenses.name
}

output "expenses_table_arn" {
    value = aws_dynamodb_table.expenses.arn
}