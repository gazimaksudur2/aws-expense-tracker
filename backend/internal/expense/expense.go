package expense

import (
	"sort"
	"strings"
	"sync"

	"github.com/gazimaksudur2/aws-expense-tracker/internal/id"
)

type Expense struct {
	ExpenseID  string `json:"expense_id"`
	UserID     string `json:"user_id,omitempty"`
	Amount     string `json:"amount"`
	Category   string `json:"category"`
	Date       string `json:"date"` // YYYY-MM-DD
	ReceiptKey string `json:"receipt_key,omitempty"`
}

// Store is implemented by MemStore (tests) and DynamoDB (Step 3).
type Store interface {
	Put(e Expense) error
	QueryMonth(userID, month string) ([]Expense, error) // month = "YYYY-MM"
}

type MemStore struct {
	mu    sync.RWMutex
	items []Expense
}

func NewMemStore() *MemStore { return &MemStore{} }

func (m *MemStore) Put(e Expense) error {
	if e.ExpenseID == "" {
		e.ExpenseID = id.New(4)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = append(m.items, e)
	return nil
}

// QueryMonth returns rows ordered like the DynamoDB sort key (date#expense_id).
func (m *MemStore) QueryMonth(userID, month string) ([]Expense, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Expense
	for _, e := range m.items {
		if e.UserID == userID && strings.HasPrefix(e.Date, month) {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Date != out[j].Date {
			return out[i].Date < out[j].Date
		}
		return out[i].ExpenseID < out[j].ExpenseID
	})
	return out, nil
}
