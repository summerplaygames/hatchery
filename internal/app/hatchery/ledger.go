package hatchery

import "container/list"

// MemLedger is a in-memory Ledger implementation that uses
// a doubly linked list to store Transactions.
type MemLedger struct {
	ledger *list.List
}

// NewMemLedger returns a new MemLedger.
func NewMemLedger() *MemLedger {
	return &MemLedger{
		ledger: list.New(),
	}
}

// Head returns the first item in the ledger.
// If the ledger is currently empty, nil is returned instead.
func (l *MemLedger) Head() *Transaction {
	if l.ledger.Len() == 0 {
		return nil
	}
	return l.ledger.Front().Value.(*Transaction)
}

// Find iterates the MemLedger until it finds a Transaction with
// an ID that matches the requested transaction ID. The second
// return parameter is whether or not a Transaction with the requested
// ID was found.
func (l *MemLedger) Find(id string) (*Transaction, bool) {
	curr := l.ledger.Front()
	for curr != nil {
		txn := curr.Value.(*Transaction)
		if txn.ID == id {
			return txn, true
		}
		curr = curr.Next()
	}
	return nil, false
}

// Append adds a Transaction to the end of the MemLedger.
func (l *MemLedger) Append(t *Transaction) {
	l.ledger.PushBack(t)
}
