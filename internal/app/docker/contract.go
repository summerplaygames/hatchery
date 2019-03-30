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
