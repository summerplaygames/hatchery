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

import "os/exec"

// PullImage pulls down a docker image using `docker pull`.
// An error is returned if the `docker pull` command fails.
func PullImage(image string) error {
	return exec.Command("docker", "pull", image).Run()
}

// Run executes a docker image with the provided command and arguments.
// The exec.Cmd for the shell process is returned. An error is returned
// if the shell processes execution fails.
func Run(image, cmd string, env map[string]string, args ...string) (*exec.Cmd, error) {
	envArr := make([]string, 0, len(env))
	for k, v := range env {
		envArr = append(envArr, "-e", k+"="+v)
	}
	arr := append(envArr, "-d", image, cmd)
	arr = append(arr, args...)
	c := exec.Command("docker", arr...)
	err := c.Run()
	return c, err
}
