For my resume, I would not build a simple "deploy my website on EC2" project. Recruiters see hundreds of those. I need something that demonstrates:

- AWS architecture design
- Serverless concepts
- Networking
- Security
- Database design
- CI/CD
- Monitoring
- Scalability
- Containers (local development)
- Automated testing


## Project Idea #1 (Highly Recommended)

# Cloud-Native Expense Tracker API on AWS

### Resume Value: ⭐⭐⭐⭐⭐

A production-style backend system similar to what companies actually build.

Story I can tell:

"Developed locally using containers and deployed serverlessly on AWS."

---

# Architecture

Phase 1 (custom JWT, Lambda authorizer):

```
                User
                 |
            API Gateway
                 |
          Lambda Authorizer  ---- checks JWT, passes user_id
                 |
          Business Lambda  ------ no token parsing
                 |
        --------------------
        |                  |
   DynamoDB            S3 Bucket
   Database            File Storage
        |
   CloudWatch

GitHub Actions --> Terraform (dev) --> AWS
```

Phase 2 (replace the authorizer, keep the business Lambda):

```
                User
                 |
            Amazon Cognito   ---- register, login, refresh tokens
                 |
               JWT
                 |
            API Gateway
                 |
          JWT Authorizer  ------ validates Cognito JWKS
                 |
          Business Lambda
                 |
        --------------------
        |                  |
   DynamoDB            S3 Bucket
```

Local development:

```
docker compose
    |
    |-- Go API (HTTP mode, same handlers as Lambda)
    |-- DynamoDB Local
    |-- LocalStack (optional, S3 only)
```

---

# What I build

A backend API where users can:

- Register/login (phase 1 in the API, phase 2 in Cognito)
- Create expenses
- Upload receipts
- Categorize spending
- View monthly reports
- Export reports


Example API:

```
POST /users/register      # phase 1 only; Cognito owns this in phase 2
POST /users/login         # phase 1 only

POST /expenses            # authorizer required
GET  /expenses/monthly

POST /receipts/upload
GET  /analytics
```

---

# Local development (Docker)

`docker-compose.yml` at the repo root.

```
Go API  -->  DynamoDB Local
                |
         LocalStack (optional profile, S3)
```

The same handler code runs two ways:

- `APP_MODE=http` inside Compose, for local curl and tests
- `APP_MODE=lambda` as the zip deployed to AWS

DynamoDB Local uses the same table names and key schema as AWS. LocalStack is off unless I am working on receipt upload. Cognito is not emulated; local phase 1 auth is the same JWT check the Lambda authorizer uses, called as HTTP middleware.

---

# AWS Services Used

## 1. AWS Lambda (Free Tier)

One Go binary, two entrypoints:

- `cmd/api` — business routes
- `cmd/authorizer` — phase 1 JWT check

I demonstrate:

✅ Serverless architecture
✅ Auto scaling
✅ Cost optimization
✅ Business logic separated from authentication


---

## 2. API Gateway

HTTP API (cheaper than REST API).

Example:

```
https://abc.execute-api.amazonaws.com/expenses
```

Protected routes use an authorizer. The business Lambda receives `user_id` and does not read the `Authorization` header.

Phase 1: Lambda authorizer. A homemade JWT has a signing secret and no JWKS URL, so API Gateway's built-in JWT authorizer cannot validate it.

Phase 2: built-in JWT authorizer pointed at the Cognito user pool issuer and app client id.

Public routes in phase 1: `POST /users/register`, `POST /users/login`.


---

## 3. Auth, two phases

### Phase 1 — custom JWT (learn the security path)

```
bcrypt password
sign JWT (secret in SSM)
Lambda authorizer verifies signature and expiry
context.user_id passed to the business Lambda
```

Users table:

```
email            partition key
user_id
name
password_hash
created_at
```

### Phase 2 — Amazon Cognito

```
User Pool
App Client
JWT access token
API Gateway JWT authorizer
```

Cognito provides registration, login, JWT, and refresh tokens. `password_hash` leaves the users table. The profile row is keyed by the Cognito `sub`. The business Lambda already reads `user_id` from authorizer context, so the swap is the authorizer and the register/login routes, not the expense code.

---

## 4. DynamoDB

### Users

Phase 1: partition key `email`. Attributes `user_id`, `name`, `password_hash`, `created_at`.

Phase 2: partition key `user_id` (Cognito `sub`). Attributes `email`, `name`, `created_at`.

### Expenses

```
user_id                 partition key
date#expense_id         sort key          example: 2026-10-10#8f3a
amount
category
receipt_key
```

`GET /expenses/monthly?month=2026-10` is one Query: partition key is the caller's `user_id`, sort key begins with `2026-10`.

I demonstrate:

- NoSQL modeling
- Partition keys and sort keys
- Query access patterns (no table scan)


---

## 5. S3

Receipt upload uses a presigned PUT URL. The client uploads straight to S3.

Object key:

```
receipts/{user_id}/{expense_id}/{filename}
```

Bucket:

- Block Public Access
- SSE-S3 encryption
- Versioning
- Lifecycle rule: expire noncurrent versions after 30 days


---

## 6. IAM Security

Two roles, least privilege.

Business Lambda:

```
dynamodb:GetItem, PutItem, Query     on the two table ARNs
s3:PutObject, s3:GetObject           on arn:aws:s3:::bucket/receipts/*
logs:CreateLogStream, PutLogEvents   on this function's log group
```

Authorizer Lambda (phase 1 only):

```
ssm:GetParameter                     on the JWT secret
logs                                 on its own log group
```

Phase 2 deletes the authorizer role and the secret. Cognito validation is done by API Gateway.

Talking point: "Implemented least privilege IAM. The function that writes expenses cannot read the signing secret."

---

## 7. CloudWatch

- Lambda logs for the business function and the authorizer
- Log retention 14 days
- Dashboard: requests/day, Lambda errors, p50 and p99 latency
- Alarm on the error metric

One dashboard stays inside the CloudWatch free allowance.

---

## 8. Testing

```
go test ./...
```

```
auth_test.go         register, login, expired token, missing token
expense_test.go      create, monthly query, user isolation
analytics_test.go    category totals for a month
```

Unit tests use a fake store. Integration tests (`-tags=integration`) run against DynamoDB Local from Compose. GitHub Actions runs both before deploy.

---

## 9. CI/CD Pipeline

```
Developer
    |
git push
    |
GitHub Actions
    |-- go test ./...
    |-- go test -tags=integration ./...   (DynamoDB Local service)
    |-- package Lambda zip (GOOS=linux GOARCH=amd64)
    |
terraform apply   (environments/dev only)
```

GitHub authenticates to AWS with OIDC. No long-lived access keys in repository secrets.

---

# Tech Stack

```
Go
Docker Compose
DynamoDB Local
AWS Lambda
API Gateway HTTP API
Lambda authorizer, then Cognito JWT authorizer
DynamoDB
S3
Terraform
GitHub Actions
CloudWatch
```

Go stays the backend language because my resume already highlights it. Docker is how I develop. AWS is where it runs.

---

# GitHub Repository Structure

```
aws-expense-tracker/

├── docker-compose.yml
├── backend/
│    ├── cmd/
│    │    ├── api/                 # business Lambda + local HTTP server
│    │    └── authorizer/          # phase 1 JWT authorizer
│    ├── internal/
│    │    ├── auth/
│    │    ├── expense/
│    │    └── analytics/
│    └── *_test.go
│
├── infrastructure/
│    └── terraform/
│         ├── modules/
│         │    ├── api/
│         │    ├── database/
│         │    ├── storage/
│         │    └── monitoring/
│         └── environments/
│              ├── dev/            # the stack I apply
│              ├── staging/        # same module, not applied yet
│              └── prod/
│
├── .github/
│    └── workflows/
│          └── deploy.yml
│
├── architecture.png
└── README.md
```

---

# Terraform

Infrastructure as Code only. No resources created by hand in the console.

```
infrastructure/terraform/
 └── environments/
       ├── dev/
       ├── staging/
       └── prod/
```

Each environment calls the same modules with a different `environment` name. Table names, the bucket, and the function include that name (`expenses-dev`). Remote state keys are separate (`expense-tracker/dev/terraform.tfstate`).

I apply **dev** only. Staging and prod exist so the layout is real: another apply is a variable change, not a rewrite. GitHub Actions targets `environments/dev`.

```
terraform -chdir=infrastructure/terraform/environments/dev apply
```

Creates:

- Lambda (api + authorizer)
- API Gateway HTTP API and authorizer
- DynamoDB tables
- IAM roles
- S3 bucket
- CloudWatch log groups, dashboard, alarm

Lambda stays **outside a VPC**. It reaches DynamoDB, S3, and SSM through the AWS public endpoints. A VPC would require a NAT gateway or interface endpoints.

---

# Free tier

Friendly:

- Lambda
- DynamoDB on-demand
- API Gateway HTTP API
- S3
- CloudWatch (one dashboard, one alarm)
- Cognito (phase 2, well inside the monthly active user allowance)
- DynamoDB Local and Docker (run on my machine)

Leave out, because they bill immediately:

- NAT Gateway
- RDS
- ECS Fargate
- Application Load Balancer
- VPC interface endpoints

Destroy the dev stack with `terraform destroy` when I am not demoing it.

---

# Resume bullets

After phase 1:

**Cloud Expense Management Platform | Go, Docker, AWS Lambda, API Gateway, DynamoDB, S3, Terraform, GitHub Actions**

- Designed and deployed a serverless expense management backend using AWS Lambda and API Gateway, achieving automatic scaling without server management.
- Developed the API locally with Docker Compose and DynamoDB Local, using the same handlers and table schema as the AWS deployment.
- Implemented DynamoDB data modeling with a user partition key and date sort key so a monthly expense report is a single Query.
- Moved JWT verification to an API Gateway Lambda authorizer so the business function receives an authenticated user id and handles expense logic only.
- Integrated Amazon S3 for private receipt storage using presigned uploads, IAM-scoped prefixes, and SSE-S3 encryption.
- Automated provisioning with Terraform across dev, staging, and prod configurations, and deployed the dev environment through GitHub Actions after `go test ./...`.
- Built a CloudWatch dashboard and error alarm for request volume, latency, and Lambda failures.

After phase 2, replace the authorizer bullet with:

- Replaced custom JWT issuance with Amazon Cognito and an API Gateway JWT authorizer, keeping the expense Lambda unchanged.

These bullets support my AWS SAA certification and my existing Go and Docker experience.
