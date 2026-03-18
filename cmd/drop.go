package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/wesgrimes/outpost/internal/grpcclient"
)

// Drop stops and discards a run.
func Drop(args []string) error {
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

	fmt.Printf("id=%s\n", droppedID)
	fmt.Println("status=dropped")

	return nil
}
