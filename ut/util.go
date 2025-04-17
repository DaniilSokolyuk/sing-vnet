package ut

import (
	"fmt"
	"os"
	"runtime"
)

func TryOrPanic[T any](f func() (T, error)) T {
	v, err := f()
	if err != nil {
		panic(err)
	}
	return v
}

func CheckRoot() bool {
	switch runtime.GOOS {
	case "windows":
		// For Windows
		_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
		if err != nil {
			fmt.Println("This program must be run as Administrator")
			return false
		}
	default:
		// For Unix-like systems (Linux, macOS)
		if os.Geteuid() != 0 {
			fmt.Println("This program must be run with sudo privileges")
			return false
		}
	}
	return true
}
