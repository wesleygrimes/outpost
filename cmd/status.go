package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/wesgrimes/outpost/internal/client"
)

// Status lists all runs or shows detail for a single run via the remote Outpost server.
func Status(args []string) {
	c, err := client.Load()
	if err != nil {
		fatalf("%v", err)
	}

	if len(args) == 0 {
		statusList(c)
	} else {
		statusDetail(c, args[0])
	}
}

func statusList(c *client.Client) {
	runs, err := c.ListRuns()
	if err != nil {
		fatalf("%v", err)
	}

	if len(runs) == 0 {
		fmt.Println("No runs found.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintln(w, "ID\tSTATUS\tMODE\tCREATED") //nolint:errcheck // writing to stdout

	for i := range runs {
		r := runs[i]
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.ID, r.Status, r.Mode, relativeTime(r.CreatedAt)) //nolint:errcheck // writing to stdout
	}

	_ = w.Flush()
}

func statusDetail(c *client.Client, id string) {
	run, err := c.GetRun(id)
	if err != nil {
		fatalf("%v", err)
	}

	fmt.Printf("id=%s\n", run.ID)
	fmt.Printf("name=%s\n", run.Name)
	fmt.Printf("mode=%s\n", run.Mode)
	fmt.Printf("status=%s\n", run.Status)
	fmt.Printf("created_at=%s\n", run.CreatedAt.Format(time.RFC3339))
	fmt.Printf("patch_ready=%t\n", run.PatchReady)

	if run.Attach != "" {
		fmt.Printf("attach=%s\n", run.Attach)
	}

	if run.Subdir != "" {
		fmt.Printf("subdir=%s\n", run.Subdir)
	}

	if run.FinalSHA != "" {
		fmt.Printf("final_sha=%s\n", run.FinalSHA)
	}

	if run.FinishedAt != nil {
		fmt.Printf("finished_at=%s\n", run.FinishedAt.Format(time.RFC3339))
	}

	if run.LogTail != "" {
		fmt.Println()
		fmt.Println("--- log tail ---")
		fmt.Println(run.LogTail)
	}
}
