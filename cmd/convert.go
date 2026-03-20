package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/wesgrimes/outpost/internal/grpcclient"
	"github.com/wesgrimes/outpost/internal/store"
)

// Convert changes a running session between interactive and headless mode.
func Convert(args []string) error {
	jsonOut, args := hasFlag(args, "--json")

	if len(args) < 2 {
		return errors.New("usage: outpost convert <run_id> <interactive|headless>")
	}

	id := args[0]
	targetStr := strings.ToLower(args[1])

	var targetMode store.Mode
	switch targetStr {
	case "interactive":
		targetMode = store.ModeInteractive
	case "headless":
		targetMode = store.ModeHeadless
	default:
		return fmt.Errorf("invalid mode %q: must be interactive or headless", targetStr)
	}

	client, err := grpcclient.Load()
	if err != nil {
		return err
	}
	defer logClose(client)

	fmt.Fprintf(os.Stderr, "converting run %s to %s...\n", id, targetMode)

	r, err := client.ConvertMode(context.Background(), id, store.ModeToProto(targetMode))
	if err != nil {
		return err
	}

	if jsonOut {
		return printJSON(map[string]string{
			"id":     r.ID,
			"mode":   string(r.Mode),
			"status": string(r.Status),
			"attach": r.Attach,
		})
	}

	printHeader()
	fmt.Println()
	printField("Run:", r.ID)
	printField("Mode:", string(r.Mode))
	printField("Status:", string(r.Status))
	if r.Attach != "" {
		printField("Attach:", r.Attach)
	}

	return nil
}
