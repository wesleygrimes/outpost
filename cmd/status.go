package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/wesgrimes/outpost/internal/grpcclient"
)

// Status lists runs or shows detail for a specific run.
func Status(args []string) error {
	follow := false
	var id string

	for _, arg := range args {
		if arg == "--follow" || arg == "-f" {
			follow = true
		} else {
			id = arg
		}
	}

	client, err := grpcclient.Load()
	if err != nil {
		return err
	}
	defer logClose(client)

	ctx := context.Background()

	if follow && id != "" {
		return followLogs(ctx, client, id)
	}

	if id != "" {
		return showRunDetail(ctx, client, id)
	}

	runs, err := client.ListRuns(ctx)
	if err != nil {
		return err
	}
	return printRunsTable(runs)
}

func followLogs(ctx context.Context, client *grpcclient.Client, id string) error {
	fmt.Fprintf(os.Stderr, "following run %s...\n", id)

	stream, err := client.TailLogs(ctx, id, true)
	if err != nil {
		return err
	}

	for {
		entry, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			fmt.Fprintln(os.Stderr, "run completed")
			return nil
		}
		if err != nil {
			return err
		}
		fmt.Println(entry.GetLine())
	}
}
