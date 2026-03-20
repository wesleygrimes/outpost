package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/wesgrimes/outpost/internal/grpcclient"
)

// Drop stops and discards a run.
func Drop(args []string) error {
	jsonOut, args := hasFlag(args, "--json")

	if len(args) < 1 {
		return errors.New("usage: outpost drop <run_id>")
	}
	id := args[0]

	client, err := grpcclient.Load()
	if err != nil {
		return err
	}
	defer logClose(client)

	droppedID, err := client.DropRun(context.Background(), id)
	if err != nil {
		return err
	}

	if jsonOut {
		return printJSON(map[string]string{
			"id":     droppedID,
			"status": "dropped",
		})
	}

	printHeader()
	fmt.Printf("\n  Dropped: %s\n", droppedID)

	return nil
}
