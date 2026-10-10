# Execution guide

This is the build order for `plan.md`. Each step should run before the next one starts. The resume bullets at the bottom of the plan match these steps.

## What you are building

A Go expense API. Locally it runs in Docker against DynamoDB Local. In AWS, API Gateway invokes Lambda, an authorizer has already checked the caller, and the function reads DynamoDB and issues S3 upload URLs. Terraform creates the AWS resources. GitHub Actions runs the tests and applies the dev environment.

Auth has two phases. Phase 1 is a JWT you sign yourself, checked by a Lambda authorizer. Phase 2 swaps that authorizer for Amazon Cognito. The expense code stays put.

## Follow a request before you write code

`POST /expenses` with a bearer token:

1. API Gateway matches the route and invokes the authorizer.
2. Phase 1: the authorizer Lambda checks the signature and expiry, then returns `user_id`. Phase 2: the built-in JWT authorizer checks the token against the Cognito user pool.
3. The business Lambda reads `user_id` from the authorizer context and writes one DynamoDB item. It does not parse the token.
4. CloudWatch stores the log line and the duration.
5. The function returns `201`.

`POST /users/register` and `POST /users/login` are public in phase 1. In phase 2 those calls go to Cognito, and your API no longer exposes them.

Receipt upload: the business Lambda returns a presigned PUT URL. The client uploads the file straight to S3.

## Why the authorizer changes shape in phase 2

API Gateway's built-in JWT authorizer validates a token using an issuer's JWKS URL and an audience. Cognito publishes that URL. A signing secret in SSM does not.

So phase 1 is a small Lambda authorizer. Phase 2 deletes it and attaches a JWT authorizer:

- Issuer: `https://cognito-idp.{region}.amazonaws.com/{userPoolId}`
- Audience: the app client id

The business handler accepts either context shape:

- Phase 1: `requestContext.authorizer.lambda.user_id`
- Phase 2: `requestContext.authorizer.jwt.claims.sub`

## Build in this order

**1. Go module and tests, in memory.** Routes:

- `POST /users/register`
- `POST /users/login`
- `POST /expenses`
- `GET /expenses/monthly`
- `POST /receipts/upload`
- `GET /analytics`

Files: `auth_test.go`, `expense_test.go`, `analytics_test.go`. Cover a bad password, an expired token, a missing token, a monthly query, one user trying to read another user's rows, and category totals.

**2. Docker Compose.**

```yaml
services:
  api:
    build: ./backend
    ports: ["8080:8080"]
    environment:
      APP_MODE: http
      DYNAMODB_ENDPOINT: http://dynamodb:8000
      USERS_TABLE: users
      EXPENSES_TABLE: expenses
      JWT_SECRET: local-dev-secret
    depends_on: [dynamodb]
  dynamodb:
    image: amazon/dynamodb-local
    ports: ["8000:8000"]
    command: ["-jar", "DynamoDBLocal.jar", "-sharedDb", "-inMemory"]
```

LocalStack is a Compose profile named `aws`, with `SERVICES=s3`, and you start it only while working on receipts. `docker compose up` alone is the API plus DynamoDB Local.

The HTTP server calls the same `Authorize` function the authorizer Lambda uses. In AWS that function is a separate entrypoint, `cmd/authorizer`.

**3. DynamoDB keys.** Create the tables from code on startup when `DYNAMODB_ENDPOINT` is set, so Compose is self-contained.

Users, phase 1: partition key `email`. Attributes `user_id`, `name`, `password_hash`, `created_at`.

Expenses: partition key `user_id`, sort key `date#expense_id` (`2026-10-10#8f3a`). Monthly query: partition key equals the caller, sort key begins with `2026-10`.

Integration tests use `-tags=integration` and hit DynamoDB Local. Plain `go test ./...` stays on the fake store so it runs without Docker.

**4. Business Lambda.** One zip from `cmd/api`, `aws-lambda-go`, API Gateway HTTP API payload v2. Handlers read `user_id` from authorizer context only.

**5. Lambda authorizer.** `cmd/authorizer` checks the HMAC JWT and returns a simple response: `isAuthorized: true` and context `{ "user_id": "..." }`. Signing secret lives in SSM. Register and login stay unauthenticated.

**6. API Gateway HTTP API.** Routes above. Authorizer on expenses, receipts, and analytics. A `$default` stage is enough; the environment name lives in the Terraform stack (`dev`), not in a second stage.

**7. S3 receipts.** Private bucket, Block Public Access, SSE-S3, versioning, noncurrent versions expire after 30 days. Keys: `receipts/{user_id}/{expense_id}/{filename}`. `POST /receipts/upload` returns a presigned PUT.

**8. IAM, two roles.**

Business role:

- `dynamodb:GetItem`, `PutItem`, `Query` on the two table ARNs
- `s3:PutObject`, `s3:GetObject` on `arn:aws:s3:::bucket/receipts/*`
- logs on its own log group

Authorizer role:

- `ssm:GetParameter` on the JWT secret
- logs on its own log group

The business function cannot read the signing secret. That split is the least-privilege sentence in an interview.

**9. CloudWatch.** Retention 14 days on both log groups. One dashboard: request count, errors, p50, p99. One alarm on errors.

**10. Terraform modules and environments.**

```text
infrastructure/terraform/
  modules/
    api/            # HTTP API, routes, authorizer, both Lambdas
    database/
    storage/
    monitoring/
  environments/
    dev/            # apply this
    staging/        # same module call, do not apply yet
    prod/
```

Each environment passes `environment = "dev"` (or staging, prod). Resource names include it: `expenses-dev`. State keys differ: `expense-tracker/dev/terraform.tfstate`.

Apply dev only:

```text
terraform -chdir=infrastructure/terraform/environments/dev apply
```

GitHub Actions uses that directory. Staging and prod are real root modules so a later environment is a backend key and a tfvars change.

**11. GitHub Actions.**

- Pull request: `go test ./...` and `go test -tags=integration ./...` with DynamoDB Local as a service container
- Push to `main`: build both Linux zips, then `terraform apply` in `environments/dev`

OIDC from GitHub to AWS. No static access keys.

**12. Phase 2, Cognito.** After dev is deployed and the phase 1 tests are green.

- User pool and app client
- Replace the Lambda authorizer with a JWT authorizer (issuer and audience above)
- Delete `password_hash`, the authorizer Lambda, its role, and the SSM secret
- Point the users table at Cognito `sub`
- Remove `POST /users/register` and `POST /users/login` from API Gateway
- Optional: a post-confirmation trigger writes the profile row

Expense handlers stay on `user_id` from authorizer context.

**13. README and diagram.** Draw both request paths (local Compose, and API Gateway → authorizer → Lambda). Document the routes, the dev API URL from Terraform outputs, and which routes are public. State that staging and prod are defined and that dev is the deployed stack.

## Resume lines and the step that earns them

| Line | Step that makes it true |
|---|---|
| Serverless, automatic scaling | API Gateway invokes Lambda per request. You pay per millisecond. |
| Docker and DynamoDB Local | Compose runs the same handlers and key schema as AWS. |
| Partition key and sort key | Expenses are queried by `user_id` and `date#expense_id`. A month is one Query. |
| Authorizer | Phase 1 Lambda authorizer. Phase 2 Cognito JWT authorizer. Business code reads `user_id` only. |
| S3, IAM, encryption | Presigned PUT, SSE-S3, business role limited to `receipts/*`, authorizer role limited to the secret. |
| Terraform environments and GitHub Actions | Modules called from dev, staging, and prod. Actions tests, then applies dev, via OIDC. |
| CloudWatch | One dashboard and one error alarm. |

## Free tier

Use Lambda, DynamoDB on-demand, API Gateway HTTP API, S3, one CloudWatch dashboard, one alarm, and Cognito in phase 2. Docker and DynamoDB Local run on your machine.

Leave out NAT Gateway, RDS, ECS Fargate, Application Load Balancer, and VPC interface endpoints. Keep both Lambdas out of a VPC so they use the AWS public endpoints for DynamoDB, S3, and SSM.

Apply the dev stack only. Run `terraform destroy` in `environments/dev` when you are not demoing.

## First session

Scaffold `backend/` with the six handlers, an in-memory store, and `auth_test.go`, `expense_test.go`, and `analytics_test.go`. Add `docker-compose.yml` and DynamoDB Local once those tests pass.
