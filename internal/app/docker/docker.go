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
func Run(image, cmd string, args ...string) (*exec.Cmd, error) {
	arr := append([]string{"-d", image, cmd}, args...)
	c := exec.Command("docker", arr...)
	err := c.Run()
	return c, err
}
