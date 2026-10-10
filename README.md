# Expense Tracker API

A small HTTP API for recording spending, attaching receipts, and reading a month back as a list and a category breakdown.

You register, log an expense, and upload the receipt directly to object storage. At the end of the month you ask for that month’s rows and for totals by category. Every expense belongs to the signed-in user. One user cannot read another user’s rows.

The same Go handlers run on your laptop and in AWS. Locally they are an HTTP server in Docker, backed by DynamoDB Local. In AWS they are a Lambda function behind API Gateway. Authentication is checked at the edge, before the function that writes expenses runs.

**Status.** This repository is the design. Application code, Terraform, and the pipeline described here are the build that follows this document.

## What you can do

| You want to | Call |
|---|---|
| Create an account and get a token | `POST /users/register`, then `POST /users/login` |
| Record a purchase | `POST /expenses` |
| Attach a photo or PDF of the receipt | `POST /receipts/upload`, then `PUT` the file to the returned URL |
| See one month of spending | `GET /expenses/monthly?month=2026-10` |
| See totals by category for that month | `GET /analytics?month=2026-10` |

Register and login are public. Every other route requires `Authorization: Bearer <token>`.

A month is the report. `GET /expenses/monthly` returns the rows. `GET /analytics` returns the sums. Both are scoped to the caller and to a single `YYYY-MM`.

### Record an expense

```http
POST /expenses
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json

{
  "amount": "42.50",
  "category": "groceries",
  "date": "2026-10-10"
}
```

```http
201 Created

{
  "expense_id": "8f3a1c",
  "amount": "42.50",
  "category": "groceries",
  "date": "2026-10-10"
}
```

`amount` is a decimal string. `date` is `YYYY-MM-DD`. `category` is a short label the client chooses (`groceries`, `rent`, `transport`, and so on).

### Attach the receipt

```http
POST /receipts/upload
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json

{
  "expense_id": "8f3a1c",
  "filename": "receipt.jpg",
  "content_type": "image/jpeg"
}
```

```http
200 OK

{
  "upload_url": "https://expense-tracker-receipts-dev.s3.amazonaws.com/receipts/...",
  "receipt_key": "receipts/user_2f8/8f3a1c/receipt.jpg",
  "expires_in": 300
}
```

The client uploads the bytes with `PUT` to `upload_url`. The API never receives the file. The URL expires in five minutes, and the object key is reserved for that user and that expense.

### Read the month

```http
GET /expenses/monthly?month=2026-10
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

```http
200 OK

{
  "month": "2026-10",
  "expenses": [
    {
      "expense_id": "8f3a1c",
      "amount": "42.50",
      "category": "groceries",
      "date": "2026-10-10",
      "receipt_key": "receipts/user_2f8/8f3a1c/receipt.jpg"
    }
  ]
}
```

```http
GET /analytics?month=2026-10
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

```http
200 OK

{
  "month": "2026-10",
  "total": "42.50",
  "by_category": [
    { "category": "groceries", "total": "42.50" }
  ]
}
```

## How a request moves

```mermaid
flowchart TD
  client[Client]
  gateway[API Gateway HTTP API]
  authorizer[Lambda authorizer]
  api[Business Lambda]
  db[(DynamoDB)]
  bucket[(S3)]
  logs[CloudWatch]

  client -->|HTTPS and bearer token| gateway
  gateway --> authorizer
  authorizer -->|user_id| gateway
  gateway --> api
  api --> db
  api -->|presigned PUT URL| client
  client -->|file bytes| bucket
  api --> logs
  authorizer --> logs
```

1. The client calls API Gateway over HTTPS with a bearer token.
2. The authorizer checks the token and returns the caller’s `user_id`. A bad or expired token stops here. The business function is not invoked.
3. The business function reads `user_id` from the authorizer context, validates the body, and writes or queries DynamoDB.
4. For a receipt, it returns a presigned URL. The client sends the file to S3.
5. CloudWatch keeps the logs, the latency, and an alarm on errors.

Register and login skip step 2. They are the only routes without an authorizer.

Locally there is no API Gateway. `docker compose` runs the same handlers as an HTTP server, and HTTP middleware calls the same `Authorize` function the authorizer Lambda uses. DynamoDB Local stores the rows. An optional Compose profile runs LocalStack when you are working on the S3 upload.

## Design

### One service, two entrypoints

`cmd/api` is the business binary. It serves HTTP when `APP_MODE=http` and the Lambda runtime when `APP_MODE=lambda`. `cmd/authorizer` is a second, smaller binary whose only job is to verify the token.

Expense rules live in one place and run unchanged on a laptop and in AWS. Authentication is a separate deployable, so a change to password hashing does not require redeploying the code that sums a month.

### The authorizer owns the token

The business function does not read `Authorization`. It trusts the `user_id` API Gateway places on the request after the authorizer succeeds.

Phase 1 signs HS256 JWTs with a secret in SSM Parameter Store. API Gateway’s built-in JWT authorizer can only validate tokens that come from an issuer with a JWKS URL, and a raw HMAC secret has no such URL. A Lambda authorizer is what can check that token.

Phase 2 replaces that function with Amazon Cognito. Cognito issues the tokens and publishes a JWKS URL. API Gateway’s JWT authorizer validates them:

- Issuer: `https://cognito-idp.{region}.amazonaws.com/{userPoolId}`
- Audience: the app client id

The business function already accepts either context shape (`authorizer.lambda.user_id`, then `authorizer.jwt.claims.sub`). Register, login, `password_hash`, the authorizer function, and the SSM secret go away. Expense queries stay as they are.

### DynamoDB access patterns

Two tables. Each query the API offers is a key lookup. Nothing scans.

**users** (phase 1). Partition key `email`, so login is a `GetItem`. Attributes: `user_id`, `name`, `password_hash`, `created_at`. Passwords are stored as bcrypt hashes.

**expenses**. Partition key `user_id`. Sort key `date#expense_id`, for example `2026-10-10#8f3a1c`.

A monthly report is one `Query`: partition key equals the caller’s `user_id`, sort key begins with `2026-10`. User isolation falls out of the key. The function never accepts a `user_id` from the body or the query string.

Analytics uses that same query and sums `amount` by `category` in the function. A month of personal expenses is a small result. A second summary table would be the next step if a single user ever produced more rows than a Query page should hold.

In phase 2 the users table swaps to partition key `user_id` (the Cognito `sub`) and drops `password_hash`. Cognito owns the email lookup.

### Receipts go straight to S3

The Lambda returns a presigned `PUT` that expires in five minutes. Object keys look like:

```text
receipts/{user_id}/{expense_id}/{filename}
```

The bucket blocks public access, encrypts with SSE-S3, versions objects, and expires noncurrent versions after 30 days. The business role may read and write only under `receipts/*`.

### Two IAM roles

| Role | Allowed to |
|---|---|
| Business Lambda | `GetItem`, `PutItem`, `Query` on the two tables. `PutObject` and `GetObject` on `receipts/*`. Write its own log group. |
| Authorizer Lambda | `GetParameter` on the JWT secret. Write its own log group. |

The function that stores an expense cannot read the signing secret. Phase 2 deletes the authorizer role with the secret. Cognito validation then happens inside API Gateway.

Both functions run outside a VPC. They call DynamoDB, S3, and SSM through AWS public endpoints. That keeps NAT gateways and interface endpoints out of the account.

### Environments

Terraform modules (`api`, `database`, `storage`, `monitoring`) are called from three root modules:

```text
infrastructure/terraform/environments/
  dev/        applied
  staging/    same modules, separate state, not applied
  prod/       same modules, separate state, not applied
```

Names carry the environment (`expenses-dev`, `expense-tracker-receipts-dev`). State keys are separate (`expense-tracker/dev/terraform.tfstate`). GitHub Actions applies `dev` only. Promoting later is a backend key and a variable, not a second copy of the resources.

GitHub authenticates to AWS with OIDC. The workflow runs `go test ./...`, then `go test -tags=integration ./...` against DynamoDB Local, builds the Linux zip, and applies `environments/dev`.

### What this account deliberately omits

Lambda, DynamoDB on-demand, API Gateway HTTP API, S3, one CloudWatch dashboard, one error alarm, and (in phase 2) Cognito. DynamoDB Local and Docker run on the developer machine.

There is no NAT gateway, RDS, ECS, Application Load Balancer, or VPC interface endpoint.

## Local development

From the repository root, once the application is in place:

```bash
docker compose up
```

That starts the API on port 8080 and DynamoDB Local on port 8000. On startup, when `DYNAMODB_ENDPOINT` is set, the process creates the tables with the same keys AWS uses.

```bash
docker compose --profile aws up
```

adds LocalStack with S3, for receipt work only.

```bash
go test ./...
go test -tags=integration ./...
```

Unit tests use an in-memory store. Integration tests are build-tagged and talk to DynamoDB Local.

| Variable | Local value |
|---|---|
| `APP_MODE` | `http` |
| `DYNAMODB_ENDPOINT` | `http://dynamodb:8000` |
| `USERS_TABLE` | `users` |
| `EXPENSES_TABLE` | `expenses` |
| `JWT_SECRET` | a dev-only secret |

## AWS layout

```text
backend/
  cmd/api/            business Lambda and local HTTP server
  cmd/authorizer/     phase 1 token check
  internal/auth/
  internal/expense/
  internal/analytics/

infrastructure/terraform/
  modules/api/
  modules/database/
  modules/storage/
  modules/monitoring/
  environments/dev/
  environments/staging/
  environments/prod/

.github/workflows/deploy.yml
docker-compose.yml
```

Apply and tear down the dev account resources with:

```bash
terraform -chdir=infrastructure/terraform/environments/dev apply
terraform -chdir=infrastructure/terraform/environments/dev destroy
```

CloudWatch holds 14 days of logs for each function, a dashboard of request count, errors, p50, and p99 latency, and an alarm on errors.

## Tests that define the behavior

| File | What it locks in |
|---|---|
| `auth_test.go` | Register, login, wrong password, missing token, expired token |
| `expense_test.go` | Create, monthly query, one user cannot read another user’s month |
| `analytics_test.go` | Category totals for a month |

## Phase 2, when phase 1 is running in dev

Cognito becomes the login system. The JWT authorizer on the HTTP API points at the user pool. The expense Lambda keeps reading `user_id` from the request context. The users table drops the password hash and uses the Cognito `sub` as its key.
