package hatchery

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
)

const (
	// ExecutionOrderParallel signifies parrellel execution of smart contracts.
	ExecutionOrderParallel = "parallel"
)

var (
	// ErrContractNotExist is returned when a request contract does not exist.
	ErrContractNotExist = errors.New("contract does not exist")
	// ErrHeapNotExist is returned when a requested heap key does not exist.
	ErrHeapNotExist = errors.New("heap value doesn't exist for key")
)

// ExecutionOrder determines how multiple instances of the same contract are executed.
type ExecutionOrder string

// Transaction is a single, atomic operation on the ledger.
type Transaction struct {
	// The transaction's unique ID.
	ID string
	// The content that is stored along with the transaction. This could
	// be the output of a smart contract or simply the payload of a
	// posted transaction.
	Content []byte `json:"-"`
}

// NewTransaction returns a new Transaction instance with the provided
// content. A unique ID is generated for the transaction.
func NewTransaction(content []byte) *Transaction {
	id := uuid.New()
	return &Transaction{
		ID:      id.String(),
		Content: content,
	}
}

// Contract is a smart contract that can be executed.
type Contract interface {
	// Execute executes the smart contract. The provided payload
	// is passed into the contract's stdin and the contract's stdout
	// is returned. An error is returned if the contract could not be
	// executed.
	Execute(payload []byte) ([]byte, error)
}

// ContractManifest contains information about a smart contract. It is used
// by a Library to track posted contracts for later execution.
type ContractManifest struct {
	// Type is the transaction type. For smart contracts, this
	// will be the name of the contract.
	Type string `json:"txn_type"`
	// Image is the Docker image that contains the contract code to be
	// executed. It should be in the format <dockerhub id>/<image name>:<image version>.
	// The docker container will be pulled down from DockerHub and the container will be
	// executed via `docker run`.
	Image string
	// Cmd is the command to execute in the smart contract's docker container.
	Cmd string
	// Args are additional application arguments that are passed in to the docker
	// container after the command.
	Args []string
	// ExecutionOrder stipulates how multiple instances of the same smart contract are
	// executed.
	ExecutionOrder ExecutionOrder `json:"execution_order"`
	// Auth is an optional DockerHub access key that is used when pulling the container image.
	// This is used when your container image is private in DockerHub.
	Auth string
}

// Library is a collection of smart contracts.
type Library interface {
	// Get returns the smart contract with the provided name.
	// If the contract doesn't exist in the library, ErrContractNotExist
	// is returned. Otherwise, an error is returned if something went wrong
	// when retrieving the contract.
	Get(name string) (Contract, error)
	// Put stores a new contract in the library, described by the provided
	// ContractManifest. An error is returned if the contract could not be
	// stored.
	Put(req *ContractManifest) error
}

// Heap is a generic key-value store that can contracts can write to to persist
// data across multiple contract executions.
type Heap interface {
	// Put inserts a key value pair in the heap. The bucket parameter is used
	// to segregate kvps into logical groups. This is useful when running multiple
	// instances of Hatchery using the same backing datastore.
	//
	// An error is returned if the kvp could not be stored.
	Put(bucket, key string, value []byte) error
	// Get retrieves a value with the provided key from the Heap. An error is
	// returned if the value for the key cannot be retrieved.
	Get(bucket string, key string) ([]byte, error)
	// GetAll returns all kvps for a bucket. An error is returned if the kvps
	// could not be retrieved.
	GetAll(bucket string) (map[string][]byte, error)
}

// Ledger is a transaction log that mimics the "blockchain."
type Ledger interface {
	// Head returns the first transaction in the ledger. This is
	// known as the "genesis" transcation. If the ledger is empty,
	// nil is returned instead.
	Head() *Transaction
	// Find searches the ledger for a transaction with the given ID and returns it.
	// I no transaction with the provided ID exists in the log, nil is returned
	// instead.
	Find(id string) *Transaction
	// Append adds a Transaction to the end of the ledger.
	Append(t *Transaction)
}

type getSCHeapRequest struct {
	Type string `json:"txn_type"`
}

// GetSCHeap returns an HTTP handler function that responds with all entries in
// the heap for a bucket.
func GetSCHeap(heap Heap) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var req getSCHeapRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// TODO: return proper response
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		h, err := heap.GetAll(req.Type)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		writeJSONResponse(w, h)
	}
}

type postTransactionRequest struct {
	Type    string `json:"txn_type"`
	Payload json.RawMessage
}

// PostTransaction returns an HTTP handler function that posts a transaction to the ledger. If
// the transaction is a smart contract, the smart contract will be executed and the output will
// be stored in the heap. Regardless, the "content" (The output in the case of a smart contract
// or the payload itself in the case of a regular transaction) is stored in a new transaction on
// the ledger.
func PostTransaction(bucket string, ledger Ledger, heap Heap, lib Library) func(http.ResponseWriter, *http.Request) {
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
		var output map[string]interface{}
		if err := json.Unmarshal(content, &output); err == nil {
			for k, v := range output {
				var buf bytes.Buffer
				if err := binary.Write(&buf, binary.BigEndian, v); err == nil {
					heap.Put(bucket, k, buf.Bytes())
				}
			}
		}
		t := NewTransaction(content)
		ledger.Append(t)
		writeJSONResponse(w, t)
	}
}

// PostContract returns an HTTP handler function that creates a new Contract in the Library.
func PostContract(lib Library) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ContractManifest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := lib.Put(&req); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}
