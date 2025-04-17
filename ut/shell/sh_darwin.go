package shell

import (
	"os"
	"os/exec"
)

func Exec(name string, args ...string) *Shell {
	command := exec.Command(name, args...)
	command.Env = os.Environ()
	return &Shell{command}
}
