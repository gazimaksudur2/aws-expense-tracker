module "github_oidc" {
  source = "../../modules/github-oidc"

  github_org      = "gazimaksudur2"
  github_repo     = "aws-expense-tracker"
  # From CloudTrail / GitHub immutable OIDC sub (owner@id / repo@id)
  github_owner_id = "107040225"
  github_repo_id  = "1412935519"
}
