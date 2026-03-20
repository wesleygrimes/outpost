package cmd

import (
	"context"
	"errors"

	"github.com/wesleygrimes/outpost/internal/grpcclient"
	"github.com/wesleygrimes/outpost/internal/ui"
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

	ui.Header("Drop " + ui.Amber(droppedID))
	ui.Errln()
	ui.Errln("  " + ui.Fail("Run "+droppedID+" stopped and discarded."))

	return nil
}
