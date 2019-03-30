package hatchery

type MemLedger struct {
	head *Transaction
	tail *Transaction
}

func (l *MemLedger) Head() *Transaction {
	return l.head
}

func (l *MemLedger) Find(id string) (*Transaction, bool) {
	curr := l.head
	for curr != nil {
		if curr.ID == id {
			return curr, true
		}
		curr = curr.Next
	}
	return nil, false
}

func (l *MemLedger) Append(t *Transaction) {
	if l.head == nil {
		l.head = t
	}
	if l.tail != nil {
		t.Prev, l.tail.Next = l.tail, t
	}
	l.tail = t
}

func (l *MemLedger) Pop() *Transaction {
	if l.tail == nil {
		return nil
	}
	tail := l.tail
	l.tail = tail.Prev
	tail.Prev = nil
	return tail
}

func (l *MemLedger) Remove(id string) (*Transaction, bool) {
	curr := l.head
	for curr != nil {
		if curr.ID == id {
			curr.Prev.Next = curr.Next
			curr.Next.Prev = curr.Prev
			curr.Next, curr.Prev = nil, nil
			return curr, true
		}
		curr = curr.Next
	}
	return nil, false
}
