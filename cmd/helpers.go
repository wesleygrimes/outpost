package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/wesgrimes/outpost/internal/grpcclient"
	"github.com/wesgrimes/outpost/internal/store"
)

func logClose(c io.Closer) {
	if err := c.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "close: %v\n", err)
	}
}

func printRunsTable(runs []*store.Run) error {
	if len(runs) == 0 {
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tSTATUS\tMODE\tCREATED")
	for _, r := range runs {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.ID, r.Status, r.Mode, r.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	return w.Flush()
}

func showRunDetail(ctx context.Context, client *grpcclient.Client, id string) error {
	r, err := client.GetRun(ctx, id)
	if err != nil {
		return err
	}
	printRunKV(r)
	return nil
}

func printRunKV(r *store.Run) {
	fmt.Printf("id=%s\n", r.ID)
	fmt.Printf("name=%s\n", r.Name)
	fmt.Printf("status=%s\n", r.Status)
	fmt.Printf("mode=%s\n", r.Mode)
	fmt.Printf("branch=%s\n", r.Branch)
	fmt.Printf("base_sha=%s\n", r.BaseSHA)
	fmt.Printf("final_sha=%s\n", r.FinalSHA)
	fmt.Printf("patch_ready=%t\n", r.PatchReady)
	fmt.Printf("created_at=%s\n", r.CreatedAt.Format("2006-01-02 15:04:05"))
	if r.FinishedAt != nil {
		fmt.Printf("finished_at=%s\n", r.FinishedAt.Format("2006-01-02 15:04:05"))
	}
	if r.Attach != "" {
		fmt.Printf("attach=%s\n", r.Attach)
	}

	if r.LogTail != "" {
		fmt.Println()
		fmt.Println("--- log tail ---")
		for line := range strings.SplitSeq(r.LogTail, "\n") {
			fmt.Println(line)
		}
	}
}
