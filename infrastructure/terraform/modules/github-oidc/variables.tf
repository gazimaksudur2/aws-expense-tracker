variable "github_org" {
  description = "GitHub org or user that owns the repository."
  type        = string
}

variable "github_repo" {
  description = "Repository name (without org)."
  type        = string
}

variable "role_name" {
  description = "IAM role name assumed by GitHub Actions."
  type        = string
  default     = "expense-tracker-github-actions"
}
