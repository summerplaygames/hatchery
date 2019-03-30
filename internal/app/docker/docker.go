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
