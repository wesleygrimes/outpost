package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/wesleygrimes/outpost/internal/config"
	"google.golang.org/grpc/status"
)

// DisplayedError wraps an error that was already shown to the user.
// main.go should exit non-zero but not print the message again.
type DisplayedError struct{ Err error }

func (e *DisplayedError) Error() string { return e.Err.Error() }
func (e *DisplayedError) Unwrap() error { return e.Err }

func logClose(c io.Closer) {
	if err := c.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "close: %v\n", err)
	}
}

// attachCmd returns the local attach command when the server is on the same
// machine, falling back to the SSH command for remote servers.
func attachCmd(attach, attachLocal string) string {
	if attachLocal != "" && isLocalServer() {
		return attachLocal
	}
	return attach
}

// isLocalServer reports whether the configured Outpost server is on the
// local machine. Returns false if the config cannot be loaded.
func isLocalServer() bool {
	cfg, err := config.LoadClient()
	if err != nil {
		return false
	}
	return cfg.IsLocalServer()
}

// humanizeGRPCError extracts the human-readable message from a gRPC error,
// stripping the "rpc error: code = ... desc = " wrapper. It unwraps through
// fmt.Errorf wrappers to find the underlying gRPC status.
func humanizeGRPCError(err error) string {
	// Unwrap until we find a gRPC status error.
	for e := err; e != nil; e = errors.Unwrap(e) {
		if s, ok := status.FromError(e); ok && s.Code() != 0 {
			msg := s.Message()
			// Strip JSON detail from capacity errors for cleaner output.
			if i := strings.Index(msg, ": ["); i > 0 {
				return msg[:i]
			}
			return msg
		}
	}
	return err.Error()
}
