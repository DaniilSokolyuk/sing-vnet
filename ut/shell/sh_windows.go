package shell

func Exec(name string, args ...string) *Shell {
	command := exec.Command(name, args...)
	command.Env = os.Environ()
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
	return &Shell{command}
}
