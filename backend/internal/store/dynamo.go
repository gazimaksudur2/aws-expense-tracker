package store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/gazimaksudur2/aws-expense-tracker/internal/auth"
	"github.com/gazimaksudur2/aws-expense-tracker/internal/expense"
	"github.com/gazimaksudur2/aws-expense-tracker/internal/id"
)

var (
	_ auth.UserStore = (*DynamoStore)(nil)
	_ expense.Store  = (*DynamoStore)(nil)
)

type userItem struct {
	Email        string `dynamodbav:"email"`
	UserID       string `dynamodbav:"user_id"`
	Name         string `dynamodbav:"name"`
	PasswordHash string `dynamodbav:"password_hash"`
	CreatedAt    string `dynamodbav:"created_at"`
}

type expenseItem struct {
	UserID     string `dynamodbav:"user_id"`
	SK         string `dynamodbav:"sk"`
	ExpenseID  string `dynamodbav:"expense_id"`
	Amount     string `dynamodbav:"amount"`
	Category   string `dynamodbav:"category"`
	Date       string `dynamodbav:"date"`
	ReceiptKey string `dynamodbav:"receipt_key,omitempty"`
}

type DynamoStore struct {
	db            *dynamodb.Client
	usersTable    string
	expensesTable string
}

func NewDynamo(ctx context.Context, endpoint, usersTable, expensesTable string) (*DynamoStore, error) {
	opts := []func(*config.LoadOptions) error{}
	if endpoint != "" {
		region := os.Getenv("AWS_REGION")
		if region == "" {
			region = "us-east-1"
		}
		opts = append(opts, config.WithRegion(region), config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("local", "local", "")))
	}
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
			o.DisableValidateResponseChecksum = true
		}
	})
	return &DynamoStore{db: client, usersTable: usersTable, expensesTable: expensesTable}, nil
}

func opCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

func (s *DynamoStore) EnsureTables(ctx context.Context) error {
	if err := s.createTable(ctx, s.usersTable, "email", ""); err != nil {
		return err
	}
	return s.createTable(ctx, s.expensesTable, "user_id", "sk")
}

func (s *DynamoStore) createTable(ctx context.Context, name, pk, sk string) error {
	attrs := []types.AttributeDefinition{
		{AttributeName: aws.String(pk), AttributeType: types.ScalarAttributeTypeS},
	}
	keys := []types.KeySchemaElement{
		{AttributeName: aws.String(pk), KeyType: types.KeyTypeHash},
	}

	if sk != "" {
		attrs = append(attrs, types.AttributeDefinition{AttributeName: aws.String(sk), AttributeType: types.ScalarAttributeTypeS})
		keys = append(keys, types.KeySchemaElement{AttributeName: aws.String(sk), KeyType: types.KeyTypeRange})
	}

	_, err := s.db.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName:            aws.String(name),
		AttributeDefinitions: attrs,
		KeySchema:            keys,
		BillingMode:          types.BillingModePayPerRequest,
	})

	var inUse *types.ResourceInUseException
	if errors.As(err, &inUse) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("create table %s: %w", name, err)
	}
	return nil
}

// auth.UserStore

func (s *DynamoStore) CreateUser(u auth.User) error {
	item, err := attributevalue.MarshalMap(userItem{
		Email:        u.Email,
		UserID:       u.UserID,
		Name:         u.Name,
		PasswordHash: u.PasswordHash,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("marshal user item: %w", err)
	}

	ctx, cancel := opCtx()
	defer cancel()
	_, err = s.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(s.usersTable),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(email)"),
	})
	var exists *types.ConditionalCheckFailedException
	if errors.As(err, &exists) {
		return auth.ErrEmailTaken
	}
	return err
}

func (s *DynamoStore) GetUserByEmail(email string) (auth.User, error) {
	ctx, cancel := opCtx()
	defer cancel()
	out, err := s.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.usersTable),
		Key:       map[string]types.AttributeValue{"email": &types.AttributeValueMemberS{Value: email}},
	})
	if err != nil {
		return auth.User{}, err
	}
	if out.Item == nil {
		return auth.User{}, auth.ErrUserNotFound
	}

	var it userItem
	if err := attributevalue.UnmarshalMap(out.Item, &it); err != nil {
		return auth.User{}, err
	}
	return auth.User{
		UserID:       it.UserID,
		Email:        it.Email,
		Name:         it.Name,
		PasswordHash: it.PasswordHash,
	}, nil
}

// expense.UserExpenseStore

func (s *DynamoStore) Put(e expense.Expense) error {
	if e.ExpenseID == "" {
		e.ExpenseID = id.New(4)
	}
	item, err := attributevalue.MarshalMap(expenseItem{
		UserID:     e.UserID,
		SK:         e.Date + "#" + e.ExpenseID,
		ExpenseID:  e.ExpenseID,
		Amount:     e.Amount,
		Category:   e.Category,
		Date:       e.Date,
		ReceiptKey: e.ReceiptKey,
	})
	if err != nil {
		return err
	}
	ctx, cancel := opCtx()
	defer cancel()
	_, err = s.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.expensesTable),
		Item:      item,
	})
	return err
}

func (s *DynamoStore) QueryMonth(userID, month string) ([]expense.Expense, error) {
	ctx, cancel := opCtx()
	defer cancel()

	pager := dynamodb.NewQueryPaginator(s.db, &dynamodb.QueryInput{
		TableName:              aws.String(s.expensesTable),
		KeyConditionExpression: aws.String("user_id = :u AND begins_with(sk, :m)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":u": &types.AttributeValueMemberS{Value: userID},
			":m": &types.AttributeValueMemberS{Value: month},
		},
	})

	var out []expense.Expense
	for pager.HasMorePages() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		var items []expenseItem
		if err := attributevalue.UnmarshalListOfMaps(page.Items, &items); err != nil {
			return nil, err
		}
		for _, item := range items {
			out = append(out, expense.Expense{
				ExpenseID:  item.ExpenseID,
				UserID:     item.UserID,
				Amount:     item.Amount,
				Category:   item.Category,
				Date:       item.Date,
				ReceiptKey: item.ReceiptKey,
			})
		}
	}
	return out, nil
}

func (s *DynamoStore) Get(userID, date, expenseID string) (expense.Expense, error) {
	ctx, cancel := opCtx()
	defer cancel()
	out, err := s.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.expensesTable),
		Key: map[string]types.AttributeValue{
			"user_id": &types.AttributeValueMemberS{Value: userID},
			"sk": &types.AttributeValueMemberS{Value: date + "#" + expenseID},
		},
	})
	if err != nil {
		return expense.Expense{}, err
	}
	if out.Item == nil {
		return expense.Expense{}, expense.ErrNotFound
	}
	var it expenseItem
	if err := attributevalue.UnmarshalMap(out.Item, &it); err != nil {
		return expense.Expense{}, err
	}

	return expense.Expense{
		ExpenseID: it.ExpenseID,
		UserID: it.UserID,
		Amount: it.Amount,
		Category: it.Category,
		Date: it.Date,
		ReceiptKey: it.ReceiptKey,
	}, nil
}