package docker

import "os/exec"

func PullImage(image string) error {
	return exec.Command("docker", "pull", image).Run()
}

func Run(image, cmd string, args ...string) (*exec.Cmd, error) {
	arr := append([]string{"-d", image, cmd}, args...)
	c := exec.Command("docker", arr...)
	err := c.Run()
	return c, err
}
