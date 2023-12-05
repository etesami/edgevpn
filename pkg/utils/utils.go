package utils

import (
	"runtime"
)

func printStackTrace() []byte {
	// Set the size of the buffer to capture stack trace
	const size = 4096
	buf := make([]byte, size)
	n := runtime.Stack(buf, false)

	// Print the stack trace
	// fmt.Printf("Stack Trace:\n%s\n", buf[:n])
	return buf[:n]
}
