//  Created on Sat Mar 30 2019
//
//  The MIT License (MIT)
//  Copyright (c) 2019 SummerPlay LLC
//
//  Permission is hereby granted, free of charge, to any person obtaining a copy of this software
//  and associated documentation files (the "Software"), to deal in the Software without restriction,
//  including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense,
//  and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so,
//  subject to the following conditions:
//
//  The above copyright notice and this permission notice shall be included in all copies or substantial
//  portions of the Software.
//
//  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED
//  TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL
//  THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
//  TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

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
