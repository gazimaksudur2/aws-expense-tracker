provider "aws" {
  region = "us-east-1"

  default_tags {
    tags = {
      Project   = "expense-tracker"
      ManagedBy = "terraform"
      Purpose   = "github-oidc"
    }
  }
}
