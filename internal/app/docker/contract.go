package docker

import (
	"fmt"
	"io/ioutil"
)

// Contract is a Contract implementation that executes Smart
// Contracts running in Docker containers.
type Contract struct {
	Name    string
	Env     map[string]string
	Image   string
	Command string
	Args    []string
}

// Execute runs the containerized smart contract by shelling out
// to `docker run`. The container's stdout is returned along with
// any errors that occur during execution.
func (c *Contract) Execute(payload []byte) ([]byte, error) {
	cmd, err := Run(c.Image, c.Command, c.Env, c.Args...)
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
