# Architecture

Living diagram for the AWS expense tracker (phase 2: Cognito). Same topology as the Mermaid block in the root README.

## Runtime (AWS)

```mermaid
flowchart LR
  client[Client]
  cognito[Cognito user pool]
  apigw[API Gateway HTTP API]
  jwt[JWT authorizer]
  lambda[Business Lambda]
  ddb[(DynamoDB expenses)]
  s3[(S3 receipts)]
  cw[CloudWatch]

  client -->|sign-up / sign-in| cognito
  cognito -->|ID token| client
  client -->|Bearer ID token| apigw
  apigw --> jwt
  jwt -->|claims.sub| apigw
  apigw --> lambda
  lambda --> ddb
  lambda -->|presigned PUT URL| client
  client -->|file bytes| s3
  lambda --> cw
```

## Local Docker

```mermaid
flowchart LR
  client[Client]
  api[API container APP_MODE=http]
  ddb[(DynamoDB Local)]
  s3[(LocalStack S3 optional)]

  client -->|register / login HS256| api
  client -->|Bearer JWT| api
  api --> ddb
  api -.->|presigned PUT if configured| s3
```

## Terraform layout

| Path | Role |
|---|---|
| `modules/database` | DynamoDB users + expenses |
| `modules/storage` | Receipts bucket |
| `modules/cognito` | User pool + app client |
| `modules/api` | Lambda + HTTP API + JWT authorizer |
| `modules/monitoring` | Logs, dashboard, alarm |
| `modules/github-oidc` | GitHub Actions deploy role |
| `environments/dev` | Day-to-day demo stack |
| `environments/staging` | Optional second stack |
| `environments/prod` | Optional long-lived stack — apply only when needed |
| `environments/oidc` | One-time OIDC provider + role |

## Auth split

| Surface | Identity |
|---|---|
| Local Docker | `POST /users/register` + `/login`, HS256 `JWT_SECRET` |
| AWS | Cognito; API Gateway JWT authorizer; Lambda reads `sub` |

See also: [architecture.svg](./architecture.svg) (static overview).
