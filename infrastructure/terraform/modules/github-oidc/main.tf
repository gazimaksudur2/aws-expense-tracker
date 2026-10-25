data "aws_caller_identity" "current" {}

# GitHub Actions OIDC provider (account-wide; safe if already exists via import).
resource "aws_iam_openid_connect_provider" "github" {
  url             = "https://token.actions.githubusercontent.com"
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = [
    "6938fd4d98bab03faadb97b34396831e3780aea1",
    "1c58a3a8518e8759bf075b76b750d4f2df264fcd",
  ]
}

data "aws_iam_policy_document" "assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRoleWithWebIdentity"]

    principals {
      type        = "Federated"
      identifiers = [aws_iam_openid_connect_provider.github.arn]
    }

    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:aud"
      values   = ["sts.amazonaws.com"]
    }

    # Limit to this repository (any branch/ref).
    # Repos created/opted-in after GitHub's immutable-sub change emit:
    #   repo:ORG@OWNER_ID/REPO@REPO_ID:ref:refs/heads/main
    # Older repos may still emit the classic form without @IDs.
    condition {
      test     = "StringLike"
      variable = "token.actions.githubusercontent.com:sub"
      values = compact([
        "repo:${var.github_org}/${var.github_repo}:*",
        var.github_owner_id != "" && var.github_repo_id != ""
        ? "repo:${var.github_org}@${var.github_owner_id}/${var.github_repo}@${var.github_repo_id}:*"
        : null,
      ])
    }
  }
}

resource "aws_iam_role" "github_actions" {
  name               = var.role_name
  assume_role_policy = data.aws_iam_policy_document.assume_role.json

  tags = {
    Project = "expense-tracker"
    Purpose = "github-actions-oidc"
  }
}

# Personal/dev sandbox: full access so Terraform can manage the stack.
# Tighten this before any shared/production account.
resource "aws_iam_role_policy_attachment" "admin" {
  role       = aws_iam_role.github_actions.name
  policy_arn = "arn:aws:iam::aws:policy/AdministratorAccess"
}
