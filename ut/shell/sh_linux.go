package shell

func Exec(name string, args ...string) *Shell {
	command := exec.Command(name, args...)
	command.Env = os.Environ()
	return &Shell{command}
}
