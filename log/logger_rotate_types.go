package log

import (
	"os"
)

const (
	defaultMaxSize = 250
	megabyte       = 1024 * 1024
)

// Rotate struct for each instance of Rotate
type Rotate struct {
	fileName        string
	rotationEnabled *bool
	maxSize         int64
	size            int64
	output          *os.File
}
