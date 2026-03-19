package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/wesgrimes/outpost/internal/config"
	"github.com/wesgrimes/outpost/internal/grpcclient"
	"github.com/wesgrimes/outpost/internal/store"
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
	printDashboard(runs)
	return nil
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

func printDashboard(runs []*store.Run) {
	// Count by status.
	var active, complete, failed, dropped int
	var running, recent []*store.Run

	for _, r := range runs {
		switch r.Status {
		case store.StatusPending, store.StatusRunning:
			active++
			running = append(running, r)
		case store.StatusComplete:
			complete++
			recent = append(recent, r)
		case store.StatusFailed:
			failed++
			recent = append(recent, r)
		case store.StatusDropped:
			dropped++
		}
	}

	cfg, cfgErr := config.LoadClient()
	serverName := ""
	if cfgErr == nil {
		serverName = cfg.Server
	}

	fmt.Printf("server=%s\n", serverName)
	fmt.Printf("active=%d\n", active)
	fmt.Printf("complete=%d\n", complete)
	fmt.Printf("failed=%d\n", failed)
	fmt.Printf("dropped=%d\n", dropped)

	for _, r := range running {
		fmt.Printf("\nrun=%s status=%s mode=%s branch=%s age=%s\n",
			r.ID, r.Status, r.Mode, r.Branch, formatAge(r.CreatedAt))
	}

	for _, r := range recent {
		age := formatAge(r.CreatedAt)
		if r.FinishedAt != nil {
			age = formatAge(*r.FinishedAt)
		}
		fmt.Printf("\nrun=%s status=%s mode=%s branch=%s age=%s patch_ready=%t\n",
			r.ID, r.Status, r.Mode, r.Branch, age, r.PatchReady)
	}
}

func formatAge(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
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
