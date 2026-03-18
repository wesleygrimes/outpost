package cmd

import (
	"fmt"
	"os"

	"github.com/wesgrimes/outpost/internal/client"
)

// KillRun terminates a running Outpost session and captures any partial work.
func KillRun(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: outpost kill <run_id>")
		fmt.Fprintln(os.Stderr, "Run 'outpost status' to find run IDs.")
		os.Exit(1)
	}

	c, err := client.Load()
	if err != nil {
		fatalf("%v", err)
	}

	run, err := c.KillRun(args[0])
	if err != nil {
		fatalf("%v", err)
	}

	fmt.Printf("id=%s\n", run.ID)
	fmt.Printf("status=%s\n", run.Status)
	fmt.Printf("patch_ready=%t\n", run.PatchReady)

	if run.PatchReady {
		fmt.Printf("\nPartial work captured. Pick up with: outpost pickup %s\n", run.ID)
	}
}
