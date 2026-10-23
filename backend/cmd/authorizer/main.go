package main

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gazimaksudur2/aws-expense-tracker/internal/auth"
)

func handler(ctx context.Context, req events.APIGatewayV2CustomAuthorizerV2Request) (events.APIGatewayV2CustomAuthorizerSimpleResponse, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" || len(req.IdentitySource) == 0 {
		return events.APIGatewayV2CustomAuthorizerSimpleResponse{IsAuthorized: false}, nil
	}
	token := strings.TrimSpace(strings.TrimPrefix(req.IdentitySource[0], "Bearer "))
	if token == "" {
		return events.APIGatewayV2CustomAuthorizerSimpleResponse{IsAuthorized: false}, nil
	}

	userID, err := auth.Authorize(token, secret)
	if err != nil {
		log.Printf("authorize failed: %v", err)
		return events.APIGatewayV2CustomAuthorizerSimpleResponse{IsAuthorized: false}, nil
	}
	return events.APIGatewayV2CustomAuthorizerSimpleResponse{
		IsAuthorized: true,
		Context: map[string]any{"user_id": userID},
	}, nil
}

func main() {
	lambda.Start(handler)
}