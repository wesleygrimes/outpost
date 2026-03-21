package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/wesleygrimes/outpost/internal/grpcclient"
	"github.com/wesleygrimes/outpost/internal/store"
	"github.com/wesleygrimes/outpost/internal/ui"
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

	ui.Errf("converting run %s to %s...\n", id, targetMode)

	r, err := client.ConvertMode(context.Background(), id, store.ModeToProto(targetMode))
	if err != nil {
		return err
	}

	attach := attachCmd(r.Attach, r.AttachLocal)

	if jsonOut {
		return printJSON(map[string]string{
			"id":     r.ID,
			"mode":   string(r.Mode),
			"status": string(r.Status),
			"attach": attach,
		})
	}

	ui.Header(fmt.Sprintf("Convert %s %s %s", ui.Amber(r.ID), ui.Dim("→"), string(r.Mode)))
	ui.Errln()
	ui.Field("Run", ui.Amber(r.ID))
	ui.Field("Mode", string(r.Mode))
	ui.Field("Status", string(r.Status))
	if attach != "" {
		ui.Field("Attach", attach)
	}

	return nil
}
