package expense

import (
	"errors"
	"sort"
	"strings"
	"sync"

	"github.com/gazimaksudur2/aws-expense-tracker/internal/id"
)

var ErrNotFound = errors.New("expense not found")

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
	Get(userID, date, expenseID string) (Expense, error)
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
	for i := range m.items {
		if m.items[i].UserID == e.UserID && m.items[i].Date == e.Date && m.items[i].ExpenseID == e.ExpenseID {
			m.items[i] = e
			return nil
		}
	}
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

func (m *MemStore) Get(userID, date, expenseID string) (Expense, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, e := range m.items {
		if e.UserID == userID && e.Date == date && e.ExpenseID == expenseID {
			return e, nil
		}
	}
	return Expense{}, ErrNotFound
}
