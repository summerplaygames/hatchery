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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/summerplaygames/hatchery/internal/app/docker"
)

// Environment keys
const (
	SCName        = "SMART_CONTRACT_NAME"
	AuthKey       = "AUTH_KEY"
	AuthID        = "AUTH_KEY_ID"
	DragonChainID = "DRAGONCHAIN_ID"
)

// Credentials are the credentials used to access the DragonChain
// API for a particular chain.
type Credentials struct {
	AuthKey       string
	AuthID        string
	DragonChainID string
}

// FSLibrary is a Library implementation that uses the filesystem.
type FSLibrary struct {
	// BasePath is the base filepath where contract manifests will be stored.
	BasePath string
	// Crednentials are the credentials used to access a DragonChain.
	Credentials Credentials

	once sync.Once
}

// Get returns the DockerContract for the given name.
// If no contract with requested name exists in the Library,
// ErrContractNotExist is returned. Otherwise, an error is returned
// only if the manifest cannot be JSON decoded.
func (l *FSLibrary) Get(name string) (Contract, error) {
	l.ensurePath()
	f, err := os.Open(filepath.Join(l.BasePath, name))
	if err != nil {
		return nil, ErrContractNotExist
	}
	defer f.Close()
	var manifest ContractManifest
	if err := json.NewDecoder(f).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("failed to read JSON manifest: %s", err)
	}
	env := map[string]string{
		SCName:        manifest.Type,
		AuthKey:       l.Credentials.AuthKey,
		AuthID:        l.Credentials.AuthID,
		DragonChainID: l.Credentials.DragonChainID,
	}
	for k, v := range manifest.Env {
		env[k] = v
	}
	return &docker.Contract{
		Name:    manifest.Type,
		Env:     env,
		Image:   manifest.Image,
		Command: manifest.Cmd,
		Args:    manifest.Args,
	}, nil
}

// Put creates a new contract defined by the provided ContractManifest.
// The image defined in the manifest is pulled down from DockerHub and the
// manfiest is stored on disk. An error is returned in the following scenarios:
//   1. The docker image could not be pulled from DockerHub.
//   2. The manifest file could not be opened for writing.
//   3. The manifest file could not be JSON encoded.
//   4. The JSON encoded manifest could not be written to disk.
func (l *FSLibrary) Put(manifest *ContractManifest) error {
	l.ensurePath()
	if err := docker.PullImage(manifest.Image); err != nil {
		return fmt.Errorf("failed to pull image: %s", err)
	}
	f, err := os.OpenFile(filepath.Join(l.BasePath, manifest.Type), os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to create manifest: %s", err)
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(manifest); err != nil {
		return fmt.Errorf("failed to write JSON manifest: %s", err)
	}
	return nil
}

func (l *FSLibrary) ensurePath() {
	l.once.Do(func() {
		os.MkdirAll(l.BasePath, 0600)
	})
}
