package hatchery

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/summerplaygames/hatchery/internal/app/docker"
)

type FSContract struct {
	Image   string
	Command string
	Args    []string
}

func (c *FSContract) Execute(payload []byte) ([]byte, error) {
	cmd, err := docker.Run(c.Image, c.Command, c.Args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute command: %s", err)
	}
	w, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to initiate pipe to stdin: %s", err)
	}
	r, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to initiate pipe from stdout: %s", err)
	}
	defer w.Close()
	if _, err := w.Write(payload); err != nil {
		return nil, fmt.Errorf("failed to pipe to stdin: %s", err)
	}
	return ioutil.ReadAll(r)
}

type FSLibrary struct {
	BasePath string
	once     sync.Once
}

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
	return &FSContract{
		Image:   manifest.Image,
		Command: manifest.Cmd,
		Args:    manifest.Args,
	}, nil
}

func (l *FSLibrary) Create(req *ContractManifest) error {
	l.ensurePath()
	if err := docker.PullImage(req.Image); err != nil {
		return fmt.Errorf("failed to pull image: %s", err)
	}
	f, err := os.OpenFile(filepath.Join(l.BasePath, req.Type), os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to create manifest: %s", err)
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(req); err != nil {
		return fmt.Errorf("failed to write JSON manifest: %s", err)
	}
	return nil
}

func (l *FSLibrary) ensurePath() {
	l.once.Do(func() {
		os.MkdirAll(l.BasePath, 0600)
	})
}
