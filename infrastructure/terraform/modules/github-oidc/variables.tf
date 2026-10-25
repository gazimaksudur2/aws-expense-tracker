variable "github_org" {
  description = "GitHub org or user that owns the repository."
  type        = string
}

variable "github_repo" {
  description = "Repository name (without org)."
  type        = string
}

variable "github_owner_id" {
  description = "Numeric GitHub owner ID for immutable OIDC sub claims (OWNER@ID). Leave empty to skip ID-qualified patterns."
  type        = string
  default     = ""
}

variable "github_repo_id" {
  description = "Numeric GitHub repository ID for immutable OIDC sub claims (REPO@ID). Leave empty to skip ID-qualified patterns."
  type        = string
  default     = ""
}

variable "role_name" {
  description = "IAM role name assumed by GitHub Actions."
  type        = string
  default     = "expense-tracker-github-actions"
}
