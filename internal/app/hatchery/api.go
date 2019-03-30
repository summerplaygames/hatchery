package hatchery

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"

	"github.com/google/uuid"
)

const (
	ExecutionOrderParallel = "parallel"
)

var (
	ErrContractNotExist = errors.New("contract does not exist")
)

type ExecutionOrder string

type Transaction struct {
	ID      string
	Prev    *Transaction `json:"-"`
	Next    *Transaction `json:"-"`
	Content []byte       `json:"-"`
}

func NewTransaction(content []byte) *Transaction {
	id := uuid.New()
	return &Transaction{
		ID:      id.String(),
		Content: content,
	}
}

type Contract interface {
	Execute(payload []byte) ([]byte, error)
}

type ContractManifest struct {
	Type           string `json:"txn_type"`
	Image          string
	Cmd            string
	Args           []string
	ExecutionOrder ExecutionOrder `json:"execution_order"`
	Auth           string
}

type Library interface {
	Get(image string) (Contract, error)
	Create(req *ContractManifest) error
}

type Heap interface {
	Put(bucket string, key, value interface{}) error
	Get(bucket string, key interface{}, typ reflect.Type) (interface{}, error)
	GetAll(bucket string) map[string]interface{}
}

type Ledger interface {
	Head() *Transaction
	Find(id string) *Transaction
	Append(t *Transaction)
	Pop() *Transaction
	Remove(id string) (*Transaction, bool)
}

type getSCHeapRequest struct {
	Type string `json:"txn_type"`
}

func getSCHeap(heap Heap) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var req getSCHeapRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// TODO: return proper response
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if h := heap.GetAll(req.Type); h != nil {
			writeJSONResponse(w, h)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}
}

type postTransactionRequest struct {
	Type    string `json:"txn_type"`
	Payload json.RawMessage
}

func postTransaction(ledger Ledger, heap Heap, lib Library) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var req postTransactionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// TODO: return proper response
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		contract, err := lib.Get(req.Type)
		if err == ErrContractNotExist {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		content, err := contract.Execute(req.Payload)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		t := NewTransaction(content)
		ledger.Append(t)
		writeJSONResponse(w, t)
	}
}

func postContract(lib Library) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ContractManifest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := lib.Create(&req); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}
