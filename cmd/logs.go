package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/wesgrimes/outpost/internal/grpcclient"
)

// Logs streams or dumps log output for a run.
func Logs(args []string) error {
	var (
		id    string
		tail  bool
		lines = 20
	)

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--tail", "-t":
			tail = true
		case "-n":
			i++
			if i >= len(args) {
				return errors.New("-n requires a value")
			}
			n, err := strconv.Atoi(args[i])
			if err != nil {
				return fmt.Errorf("invalid -n value: %w", err)
			}
			lines = n
		default:
			id = args[i]
		}
	}

	if id == "" {
		return errors.New("usage: outpost logs <run_id> [--tail] [-n lines]")
	}

	client, err := grpcclient.Load()
	if err != nil {
		return err
	}
	defer logClose(client)

	ctx := context.Background()

	if tail {
		return tailLogs(ctx, client, id, lines)
	}
	return dumpLogs(ctx, client, id, lines)
}

// dumpLogs fetches log lines (non-follow) and prints the last N.
func dumpLogs(ctx context.Context, client *grpcclient.Client, id string, lines int) error {
	stream, err := client.TailLogs(ctx, id, false)
	if err != nil {
		return err
	}

	// Collect all lines, keep last N using a ring buffer.
	buf := make([]string, 0, lines)
	for {
		entry, recvErr := stream.Recv()
		if errors.Is(recvErr, io.EOF) {
			break
		}
		if recvErr != nil {
			return recvErr
		}
		buf = append(buf, entry.GetLine())
	}

	// Print last N lines.
	start := 0
	if len(buf) > lines {
		start = len(buf) - lines
	}
	for _, line := range buf[start:] {
		fmt.Println(line)
	}
	return nil
}

// tailLogs prints the last N lines then follows.
func tailLogs(ctx context.Context, client *grpcclient.Client, id string, lines int) error {
	// First pass: collect last N lines (non-follow).
	stream, err := client.TailLogs(ctx, id, false)
	if err != nil {
		return err
	}

	buf := make([]string, 0, lines)
	for {
		entry, recvErr := stream.Recv()
		if errors.Is(recvErr, io.EOF) {
			break
		}
		if recvErr != nil {
			return recvErr
		}
		buf = append(buf, entry.GetLine())
	}

	start := 0
	if len(buf) > lines {
		start = len(buf) - lines
	}
	for _, line := range buf[start:] {
		fmt.Println(line)
	}

	// Second pass: follow from current position.
	fmt.Fprintf(os.Stderr, "--- following ---\n")
	followStream, err := client.TailLogs(ctx, id, true)
	if err != nil {
		return err
	}

	for {
		entry, recvErr := followStream.Recv()
		if errors.Is(recvErr, io.EOF) {
			fmt.Fprintln(os.Stderr, "run completed")
			return nil
		}
		if recvErr != nil {
			return recvErr
		}
		fmt.Println(entry.GetLine())
	}
}
