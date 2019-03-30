package hatchery

import "container/list"

type MemLedger struct {
	ledger *list.List
}

func NewMemLedger() *MemLedger {
	return &MemLedger{
		ledger: list.New(),
	}
}

func (l *MemLedger) Head() *Transaction {
	return l.ledger.Front().Value.(*Transaction)
}

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

func (l *MemLedger) Append(t *Transaction) {
	l.ledger.PushBack(t)
}

func (l *MemLedger) Pop() *Transaction {
	if l.ledger.Len() == 0 {
		return nil
	}
	return l.ledger.Remove(l.ledger.Back()).(*Transaction)
}

func (l *MemLedger) Remove(id string) (*Transaction, bool) {
	curr := l.ledger.Front()
	for curr != nil {
		txn := curr.Value.(*Transaction)
		if txn.ID == id {
			l.ledger.Remove(curr)
			return txn, true
		}
		curr = curr.Next()
	}
	return nil, false
}
