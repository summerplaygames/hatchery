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

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/mux"

	"github.com/google/uuid"
)

const (
	// ExecutionOrderParallel signifies parrellel execution of smart contracts.
	ExecutionOrderParallel = "parallel"
	// ExecutionOrderSerial signifies serial execution of smart contracts.
	ExecutionOrderSerial = "serial"
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
	// Args are optional additional application arguments that are passed in to the docker
	// container after the command.
	Args []string
	// ExecutionOrder stipulates how multiple instances of the same smart contract are
	// executed. Valid values are ExecutionOrderParallel and ExecutionOrderSerial.
	ExecutionOrder ExecutionOrder `json:"execution_order"`
	// Env is an optional set of environment variables to pass into the contract at runtime.
	Env map[string]string
	// Cron is an optional rate of scheduled execution specified as a cron.
	Cron string
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

type postTransactionRequest struct {
	Type    string `json:"txn_type"`
	Payload json.RawMessage
}

// Application contains of all of the application state and its dependencies.
type Application struct {
	Bucket  string
	Heap    Heap
	Ledger  Ledger
	Lib     Library
	cronMu  sync.Mutex
	cronTab map[string]*CronJob
	once    sync.Once
}

// SetupRoutes initializes the HTTP routes with the provided muxer.
func (a *Application) SetupRoutes(muxer *mux.Router) {
	muxer.HandleFunc("/get/{sc_name}/{key}", a.GetSCHeap()).Methods(http.MethodGet)
	muxer.HandleFunc("/transaction", a.PostTransaction()).Methods(http.MethodPost)
	muxer.HandleFunc("/contract", a.PostContract()).Methods(http.MethodPost)
}

// Shutdown shuts down the application. All currently running cron jobs will be stopped.
func (a *Application) Shutdown() {
	a.cronMu.Lock()
	defer a.cronMu.Unlock()
	for _, cron := range a.cronTab {
		cron.Stop()
	}
}

// GetSCHeap returns an HTTP handler function that responds with the heap data for the requested
// contract and key.
func (a *Application) GetSCHeap() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		name := vars["sc_name"]
		key := vars["key"]
		h, err := a.Heap.Get(name, key)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		writeJSONResponse(w, h)
	}
}

// PostTransaction returns an HTTP handler function that posts a transaction to the ledger. If
// the transaction is a smart contract, the smart contract will be executed and the output will
// be stored in the heap. Regardless, the "content" (The output in the case of a smart contract
// or the payload itself in the case of a regular transaction) is stored in a new transaction on
// the ledger.
func (a *Application) PostTransaction() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var req postTransactionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// TODO: return proper response
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		contract, err := a.Lib.Get(req.Type)
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
					a.Heap.Put(a.Bucket, k, buf.Bytes())
				}
			}
		}
		t := NewTransaction(content)
		a.Ledger.Append(t)
		writeJSONResponse(w, t)
	}
}

// PostContract returns an HTTP handler function that creates a new Contract in the Library.
// If the request specifies a cron interval, a new cron job is started in the background.
func (a *Application) PostContract() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ContractManifest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var interval time.Duration
		if req.Cron != "" {
			interval, err = time.ParseDuration(req.Cron)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}
		if err := a.Lib.Put(&req); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if interval > 0 {
			a.startCronJob(w, req.Type, interval)
		}
	}
}

func (a *Application) startCronJob(w http.ResponseWriter, name string, interval time.Duration) {
	a.ensureCronTab()
	contract, err := a.Lib.Get(name)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	cron := NewCronJob(interval, contract)
	// In order to properly start the cron job, we need to aggressively consume the errros,
	// aggressively consume the output, and finally, start the cron job itself.
	go func() {
		for err := range cron.Errors() {
			fmt.Fprintln(os.Stderr, err)
		}
	}()
	go func() {
		for result := range cron.Output() {
			fmt.Println(result)
		}
	}()
	go func() {
		if err := cron.Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}()
	a.cronMu.Lock()
	a.cronTab[name] = cron
	a.cronMu.Unlock()
}

func (a *Application) ensureCronTab() {
	a.once.Do(func() {
		a.cronTab = make(map[string]*CronJob)
	})
}
