terraform {
    required_version = ">= 1.5"

    required_providers {
        aws = {
            source = "hashicorp/aws"
            Version = "~> 5.0"
        }
    }

    backend "s3" {
        bucket = "expense-tracker-tfstate-611728563978"
        key = "expense-tracker/dev/terraform.tfstate"
        region = "us-east-1"
    }
}