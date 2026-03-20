package cmd

import (
	"fmt"
	"io"
	"os"
)

func logClose(c io.Closer) {
	if err := c.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "close: %v\n", err)
	}
}
